using System;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200014B RID: 331
	public class MessageEventArgs : EventArgs
	{
		// Token: 0x17000302 RID: 770
		// (get) Token: 0x06000BAD RID: 2989 RVA: 0x00039D88 File Offset: 0x00037F88
		// (set) Token: 0x06000BAE RID: 2990 RVA: 0x00039D90 File Offset: 0x00037F90
		public string Message { get; set; }

		// Token: 0x06000BAF RID: 2991 RVA: 0x00039D99 File Offset: 0x00037F99
		public MessageEventArgs(string message)
		{
			this.Message = message;
		}
	}
}
