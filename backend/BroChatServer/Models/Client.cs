using System.Net.WebSockets;

namespace BroChatServer.Models
{
    public class Client
    {
        public string ID { get; set; } = string.Empty;
        public string Recipient { get; set; } = string.Empty;
        private readonly SemaphoreSlim _sendLock = new(1,1);
        private readonly CancellationTokenSource _cancellationToken = new();
        
        private WebSocket? _webSocket;
        public WebSocket? WebSocketConnection 
        { 
            get => _webSocket; 
            set => _webSocket = value; 
        }
        public Action<Client>? OnDisconnect { get; set; }

        public async ValueTask DisconnectAsync(string reason = "Normal Closure")
        {
            // Atomic swap the websocket to null
            var socket = Interlocked.Exchange(ref _webSocket, null);

            if (socket == null)
            {
                return;
            }

            try
            {
                // Cancel the token if not canceled yet
                if (!_cancellationToken.IsCancellationRequested)
                {
                    _cancellationToken.Cancel();
                }

                // Close if open
                if (socket.State == WebSocketState.Open
                || socket.State == WebSocketState.CloseReceived)
                {
                    using var closeCts = new CancellationTokenSource(TimeSpan.FromSeconds(3));
                    await socket.CloseAsync(WebSocketCloseStatus.NormalClosure, reason, closeCts.Token).ConfigureAwait(false);
                }
            }
            catch
            {
                
            }
            finally
            {
                socket.Dispose();
                OnDisconnect?.Invoke(this);
            }
        }

        public async ValueTask SendMessageAsync(ReadOnlyMemory<byte> message, Action<Client>? onDeliveryFailed = null, Client? sender = null)
        {
            if (message.IsEmpty)
            {
                return;
            }

            try {
                bool acquired = await _sendLock.WaitAsync(TimeSpan.FromSeconds(5), _cancellationToken.Token).ConfigureAwait(false);
                if (!acquired)
                {
                    if (onDeliveryFailed != null && sender != null) onDeliveryFailed(sender);
                    _ = DisconnectAsync("Send Timeout");
                    return;
                }

                try
                {
                    if (_webSocket is {State: WebSocketState.Open} socket)
                    {
                        await socket.SendAsync(
                            message,
                            WebSocketMessageType.Text,
                            true,
                            _cancellationToken.Token
                        ).ConfigureAwait(false);
                    }
                    else
                    {
                        if (onDeliveryFailed != null && sender != null) onDeliveryFailed(sender);
                    }
                }
                finally
                {
                    _sendLock.Release();
                }
            } 
            catch (OperationCanceledException)
            {
                if (onDeliveryFailed != null && sender != null) onDeliveryFailed(sender);
            }
            catch (Exception)
            {
                if (onDeliveryFailed != null && sender != null) onDeliveryFailed(sender);
                _ = DisconnectAsync("Network error during send!");
            }
        }
    }
}