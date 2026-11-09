using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200012F RID: 303
	public class DimensionClassOption
	{
		// Token: 0x170002D2 RID: 722
		// (get) Token: 0x06000AC0 RID: 2752 RVA: 0x00034E28 File Offset: 0x00033028
		// (set) Token: 0x06000AC1 RID: 2753 RVA: 0x00034E30 File Offset: 0x00033030
		public string Name { get; set; }

		// Token: 0x170002D3 RID: 723
		// (get) Token: 0x06000AC2 RID: 2754 RVA: 0x00034E39 File Offset: 0x00033039
		// (set) Token: 0x06000AC3 RID: 2755 RVA: 0x00034E41 File Offset: 0x00033041
		public string Description { get; set; }

		// Token: 0x06000AC4 RID: 2756 RVA: 0x00034E4A File Offset: 0x0003304A
		public DimensionClassOption(string name, string description)
		{
			this.Name = name;
			this.Description = description;
		}
	}
}
