using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x0200013A RID: 314
	public class FunctionSelectedModel
	{
		// Token: 0x06000AFE RID: 2814 RVA: 0x000356B1 File Offset: 0x000338B1
		public FunctionSelectedModel()
		{
			this.Type = DefinitionType.NULL;
		}

		// Token: 0x170002E0 RID: 736
		// (get) Token: 0x06000AFF RID: 2815 RVA: 0x000356C0 File Offset: 0x000338C0
		// (set) Token: 0x06000B00 RID: 2816 RVA: 0x000356C8 File Offset: 0x000338C8
		public PackageKey ProgramKey { get; set; }

		// Token: 0x170002E1 RID: 737
		// (get) Token: 0x06000B01 RID: 2817 RVA: 0x000356D1 File Offset: 0x000338D1
		// (set) Token: 0x06000B02 RID: 2818 RVA: 0x000356D9 File Offset: 0x000338D9
		public DefinitionType Type { get; set; }

		// Token: 0x170002E2 RID: 738
		// (get) Token: 0x06000B03 RID: 2819 RVA: 0x000356E2 File Offset: 0x000338E2
		// (set) Token: 0x06000B04 RID: 2820 RVA: 0x000356EA File Offset: 0x000338EA
		public string Name { get; set; }

		// Token: 0x170002E3 RID: 739
		// (get) Token: 0x06000B05 RID: 2821 RVA: 0x000356F3 File Offset: 0x000338F3
		// (set) Token: 0x06000B06 RID: 2822 RVA: 0x000356FB File Offset: 0x000338FB
		public string Target { get; set; }

		// Token: 0x170002E4 RID: 740
		// (get) Token: 0x06000B07 RID: 2823 RVA: 0x00035704 File Offset: 0x00033904
		// (set) Token: 0x06000B08 RID: 2824 RVA: 0x0003570C File Offset: 0x0003390C
		public FunctionSelectedModel Child { get; set; }
	}
}
