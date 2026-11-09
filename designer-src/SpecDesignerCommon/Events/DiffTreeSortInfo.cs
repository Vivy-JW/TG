using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000021 RID: 33
	public class DiffTreeSortInfo
	{
		// Token: 0x17000036 RID: 54
		// (get) Token: 0x060000FF RID: 255 RVA: 0x00005E4E File Offset: 0x0000404E
		// (set) Token: 0x06000100 RID: 256 RVA: 0x00005E56 File Offset: 0x00004056
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000037 RID: 55
		// (get) Token: 0x06000101 RID: 257 RVA: 0x00005E5F File Offset: 0x0000405F
		// (set) Token: 0x06000102 RID: 258 RVA: 0x00005E67 File Offset: 0x00004067
		public string NodeName { get; set; }

		// Token: 0x17000038 RID: 56
		// (get) Token: 0x06000103 RID: 259 RVA: 0x00005E70 File Offset: 0x00004070
		// (set) Token: 0x06000104 RID: 260 RVA: 0x00005E78 File Offset: 0x00004078
		public int SortIndex { get; set; }
	}
}
