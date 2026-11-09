using System;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x02000020 RID: 32
	public class CDataEventArgs : EventArgs
	{
		// Token: 0x1700001A RID: 26
		// (get) Token: 0x0600006B RID: 107 RVA: 0x00002A4E File Offset: 0x00000C4E
		// (set) Token: 0x0600006C RID: 108 RVA: 0x00002A56 File Offset: 0x00000C56
		public string CDATA { get; private set; }

		// Token: 0x0600006D RID: 109 RVA: 0x00002A5F File Offset: 0x00000C5F
		public CDataEventArgs(string cdata)
		{
			this.CDATA = cdata;
		}
	}
}
