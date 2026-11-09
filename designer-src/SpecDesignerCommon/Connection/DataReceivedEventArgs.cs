using System;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200006B RID: 107
	public class DataReceivedEventArgs : EventArgs
	{
		// Token: 0x0600040A RID: 1034 RVA: 0x00012B2D File Offset: 0x00010D2D
		public DataReceivedEventArgs(string data)
		{
			this.Data = data;
		}

		// Token: 0x1700010F RID: 271
		// (get) Token: 0x0600040B RID: 1035 RVA: 0x00012B3C File Offset: 0x00010D3C
		// (set) Token: 0x0600040C RID: 1036 RVA: 0x00012B44 File Offset: 0x00010D44
		public string Data { get; private set; }
	}
}
