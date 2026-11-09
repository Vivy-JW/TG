using System;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x02000021 RID: 33
	public class DeclarationEventArgs : EventArgs
	{
		// Token: 0x1700001B RID: 27
		// (get) Token: 0x0600006E RID: 110 RVA: 0x00002A6E File Offset: 0x00000C6E
		// (set) Token: 0x0600006F RID: 111 RVA: 0x00002A76 File Offset: 0x00000C76
		public string Declaration { get; private set; }

		// Token: 0x06000070 RID: 112 RVA: 0x00002A7F File Offset: 0x00000C7F
		public DeclarationEventArgs(string dec)
		{
			this.Declaration = dec;
		}
	}
}
