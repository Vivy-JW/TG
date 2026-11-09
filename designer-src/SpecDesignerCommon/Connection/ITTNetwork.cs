using System;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200005B RID: 91
	public interface ITTNetwork
	{
		// Token: 0x170000B9 RID: 185
		// (get) Token: 0x06000304 RID: 772
		// (set) Token: 0x06000305 RID: 773
		string LoginPrompt { get; set; }

		// Token: 0x170000BA RID: 186
		// (get) Token: 0x06000306 RID: 774
		// (set) Token: 0x06000307 RID: 775
		string PasswordPrompt { get; set; }

		// Token: 0x06000308 RID: 776
		bool Connect(string username, string password);

		// Token: 0x06000309 RID: 777
		int WaitFor(params string[] keyword);

		// Token: 0x0600030A RID: 778
		void Send(string msg);

		// Token: 0x0600030B RID: 779
		void SendUTF8(string msg);

		// Token: 0x170000BB RID: 187
		// (get) Token: 0x0600030C RID: 780
		bool IsConnected { get; }

		// Token: 0x0600030D RID: 781
		void Close();

		// Token: 0x1400000B RID: 11
		// (add) Token: 0x0600030E RID: 782
		// (remove) Token: 0x0600030F RID: 783
		event EventHandler<DataReceivedEventArgs> OnDataReceived;
	}
}
