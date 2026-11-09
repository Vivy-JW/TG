using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x0200005A RID: 90
	public class ConnectionStatusChangedEventArgs
	{
		// Token: 0x170000B5 RID: 181
		// (get) Token: 0x060002FB RID: 763 RVA: 0x0000C98E File Offset: 0x0000AB8E
		// (set) Token: 0x060002FC RID: 764 RVA: 0x0000C996 File Offset: 0x0000AB96
		public bool IsLogin { get; set; }

		// Token: 0x170000B6 RID: 182
		// (get) Token: 0x060002FD RID: 765 RVA: 0x0000C99F File Offset: 0x0000AB9F
		// (set) Token: 0x060002FE RID: 766 RVA: 0x0000C9A7 File Offset: 0x0000ABA7
		public string ConnectionArea { get; set; }

		// Token: 0x170000B7 RID: 183
		// (get) Token: 0x060002FF RID: 767 RVA: 0x0000C9B0 File Offset: 0x0000ABB0
		// (set) Token: 0x06000300 RID: 768 RVA: 0x0000C9B8 File Offset: 0x0000ABB8
		public string LoginName { get; set; }

		// Token: 0x170000B8 RID: 184
		// (get) Token: 0x06000301 RID: 769 RVA: 0x0000C9C1 File Offset: 0x0000ABC1
		// (set) Token: 0x06000302 RID: 770 RVA: 0x0000C9C9 File Offset: 0x0000ABC9
		public string ServerIP { get; set; }
	}
}
