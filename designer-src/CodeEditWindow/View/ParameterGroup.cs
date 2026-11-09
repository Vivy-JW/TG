using System;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x0200003B RID: 59
	public class ParameterGroup
	{
		// Token: 0x1700005A RID: 90
		// (get) Token: 0x0600027A RID: 634 RVA: 0x000139B1 File Offset: 0x00011BB1
		// (set) Token: 0x0600027B RID: 635 RVA: 0x000139B9 File Offset: 0x00011BB9
		public string SystemModule { get; set; }

		// Token: 0x1700005B RID: 91
		// (get) Token: 0x0600027C RID: 636 RVA: 0x000139C2 File Offset: 0x00011BC2
		// (set) Token: 0x0600027D RID: 637 RVA: 0x000139CA File Offset: 0x00011BCA
		public string Local { get; set; }

		// Token: 0x1700005C RID: 92
		// (get) Token: 0x0600027E RID: 638 RVA: 0x000139D3 File Offset: 0x00011BD3
		// (set) Token: 0x0600027F RID: 639 RVA: 0x000139DB File Offset: 0x00011BDB
		public string SelfDefVariable { get; set; }
	}
}
