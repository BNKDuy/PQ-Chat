using System.Collections.Concurrent;
using System.Net.WebSockets;
using System.Reflection.Metadata;
using System.Threading.Channels;
using Microsoft.VisualBasic;

namespace BroChatServer.Models
{
    public abstract class HubCommand {}
    public class RegisterCommand : HubCommand {public Client Client { get; init; } = null!;}
    public class UnregisterCommand : HubCommand {public Client Client { get; init; } = null!;}
    public class MessageCommand : HubCommand {public ClientToHubMessage Packet { get; init; } = null!;}

    public class ChatHub
    {
        private readonly Dictionary<string, Client> _Connections = new();
        private readonly Channel<HubCommand> _HubChannel = Channel.CreateUnbounded<HubCommand>();

        public async ValueTask RegisterClient(Client client) => await _HubChannel.Writer.WriteAsync(new RegisterCommand{Client = client});
        public async ValueTask UnregisterClient(Client client) => await _HubChannel.Writer.WriteAsync(new UnregisterCommand{Client = client});
        public async ValueTask SendMessage(ClientToHubMessage packet) => await _HubChannel.Writer.WriteAsync(new MessageCommand{Packet = packet});

        public static readonly byte[] ErrUserMoreThanOneConnection = 
            """{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"You can only have one connection at a time"}"""u8.ToArray();

        public static readonly byte[] ErrUserOffline = 
            """{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"The message was not delivered (the recipient is offline)."}"""u8.ToArray();

        public static readonly byte[] ErrDeliveryFailed = 
            """{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"The message was not delivered (the recipient cannot recieve any message now). Please try again later."}"""u8.ToArray();

        public static readonly byte[] MsgStartMLKEM = 
            """{"From":"SERVER","To":"YOU","Type":"MLKEM_INIT","Content":"Start ML-KEM"}"""u8.ToArray();

        public static readonly byte[] MsgStopEncryptedSession = 
            """{"From":"SERVER","To":"YOU","Type":"STOP_ENCRYPTED_SESSION","Content":"Stop encrypted session."}"""u8.ToArray();

        private readonly Action<Client> _deliveryFailedCallback = static sender => 
        {
            _ = sender.SendMessageAsync(ErrDeliveryFailed);
        };

        private void OnConnectionClose(Client client)
        {
            _ = UnregisterClient(client);
        }

        public async Task RunHub(CancellationToken cancellationToken = default) 
        {
            await foreach (var command in _HubChannel.Reader.ReadAllAsync(cancellationToken))
            {
                switch (command) 
                {
                    case RegisterCommand registerCommand:
                    HandleRegisterClient(registerCommand.Client);
                    break;
                    case UnregisterCommand unregisterCommand:
                    HandleUnregisterClient(unregisterCommand.Client);
                    break;
                    case MessageCommand messageCommand:
                    HandleChatCommand(messageCommand.Packet);
                    break;
                }
            }
        }

        private void HandleRegisterClient(Client client)
        {
            if (_Connections.TryGetValue(client.ID, out var prevClient))
            {
                _ = prevClient.DisconnectAsync();
            }
            _Connections[client.ID] = client;
            client.OnDisconnect = OnConnectionClose;
            var recipient = client.Recipient;

            if (_Connections.TryGetValue(recipient, out var recipientConnection))
            {
                _ = recipientConnection.SendMessageAsync(MsgStartMLKEM);
            }
        }

        private void HandleUnregisterClient(Client client)
        {
            if (_Connections.TryGetValue(client.ID, out var curClient))
            {
                _ = curClient.DisconnectAsync();
                _Connections.Remove(client.ID);
                if (_Connections.TryGetValue(client.Recipient, out var recipient)) 
                {
                    _ = recipient.SendMessageAsync(MsgStopEncryptedSession);
                }
            }
        }

        private void HandleChatCommand(ClientToHubMessage packet)
        {
            if (!_Connections.TryGetValue(packet.To, out var recipient)
            || recipient == null)
            {
                _ = packet.From.SendMessageAsync(ErrUserOffline);
                return;
            }
            _ = recipient.SendMessageAsync(packet.Content, _deliveryFailedCallback, packet.From);
        }
    }
}