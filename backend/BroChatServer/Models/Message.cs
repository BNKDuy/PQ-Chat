namespace BroChatServer.Models
{
    public class ClientToHubMessage
    {
        public Client From { init; get; }= null!;
        public string To { init; get; } = "";
        public byte[] Content { init; get; } = Array.Empty<byte>();       
    }
}