using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200012E RID: 302
	public class DimensionOption
	{
		// Token: 0x170002CF RID: 719
		// (get) Token: 0x06000AB9 RID: 2745 RVA: 0x00034DD4 File Offset: 0x00032FD4
		// (set) Token: 0x06000ABA RID: 2746 RVA: 0x00034DDC File Offset: 0x00032FDC
		public string No { get; set; }

		// Token: 0x170002D0 RID: 720
		// (get) Token: 0x06000ABB RID: 2747 RVA: 0x00034DE5 File Offset: 0x00032FE5
		// (set) Token: 0x06000ABC RID: 2748 RVA: 0x00034DED File Offset: 0x00032FED
		public string Description { get; set; }

		// Token: 0x170002D1 RID: 721
		// (get) Token: 0x06000ABD RID: 2749 RVA: 0x00034DF6 File Offset: 0x00032FF6
		// (set) Token: 0x06000ABE RID: 2750 RVA: 0x00034DFE File Offset: 0x00032FFE
		public List<DimensionClassOption> Class { get; set; }

		// Token: 0x06000ABF RID: 2751 RVA: 0x00034E07 File Offset: 0x00033007
		public DimensionOption(string no, string description)
		{
			this.No = no;
			this.Description = description;
			this.Class = new List<DimensionClassOption>();
		}
	}
}
