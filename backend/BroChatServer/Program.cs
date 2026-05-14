using System.Buffers;
using System.Net.WebSockets;
using BroChatServer.Models;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
// Learn more about configuring OpenAPI at https://aka.ms/aspnet/openapi
builder.Services.AddOpenApi();
builder.Services.AddSingleton<ChatHub>();
var app = builder.Build();

var chatHub = app.Services.GetRequiredService<ChatHub>();
_ = Task.Run(() => chatHub.RunHub(app.Lifetime.ApplicationStopping));

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseHttpsRedirection();

app.UseWebSockets();

app.Map("/ws", async context =>
{
    if (context.WebSockets.IsWebSocketRequest) {
        var id = context.Request.Query["id"].ToString();
        var recipient = context.Request.Query["recipient"].ToString();

        if (string.IsNullOrEmpty(id) || string.IsNullOrEmpty(recipient))
        {
            context.Response.StatusCode = 400; // Bad Request
            return;
        }

        var websocket = await context.WebSockets.AcceptWebSocketAsync();
        if (websocket == null)
        {
            return;
        }

        var client = new Client
        {
            ID = id,
            Recipient = recipient,
            WebSocketConnection = websocket
        };

        await chatHub.RegisterClient(client);

        byte[] rentedArray = ArrayPool<byte>.Shared.Rent(1024 * 4);
        Memory<byte> bufferMemory = rentedArray;
        try
        {
            while (websocket.State == WebSocketState.Open)
            {
                var result = await websocket.ReceiveAsync(bufferMemory, CancellationToken.None);

                if (result.MessageType == WebSocketMessageType.Close)
                {
                    break; // The client requested to close the connection. Break the loop.
                }

                if (!result.EndOfMessage)
                {
                    await client.DisconnectAsync("Message exceeded 4KB limit");
                    break;
                }

                if (result.Count > 0) 
                {
                    // Copy the exact bytes we just received
                    var content = bufferMemory.Slice(0, result.Count).ToArray();

                    // Create the command for the Hub
                    var packet = new ClientToHubMessage 
                    {
                        From = client,
                        To = client.Recipient,
                        Content = content 
                    };

                    // Push the message into the Channel (Thread-safe!)
                    await chatHub.SendMessage(packet);
                }
            }
        }
        catch (WebSocketException)
        {
            // This happens when the user's internet physically drops. 
            // We can safely swallow this exception because the finally block handles cleanup.
        }
        catch (Exception ex)
        {
            // Catch-all for unexpected crashes during the receive loop
            Console.WriteLine($"Error receiving from {client.ID}: {ex.Message}");
        }
        finally
        {
            // 5. The Grand Finale: Disconnect and Unregister
            // This calls client.DisconnectAsync(), which swaps the socket to null, closes it, 
            // and invokes OnDisconnect, safely telling the ChatHub to remove them!
            ArrayPool<byte>.Shared.Return(rentedArray);
            await client.DisconnectAsync("Receive Loop Ended");
        }
    }
    else
    {
        context.Response.StatusCode = 400;
    }

    

    

});

app.Run();