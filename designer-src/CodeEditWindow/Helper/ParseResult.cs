using System;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000003 RID: 3
	public class ParseResult
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000003 RID: 3 RVA: 0x000020D8 File Offset: 0x000002D8
		// (set) Token: 0x06000004 RID: 4 RVA: 0x000020E0 File Offset: 0x000002E0
		public int StartLineNumber { get; set; }

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000005 RID: 5 RVA: 0x000020E9 File Offset: 0x000002E9
		// (set) Token: 0x06000006 RID: 6 RVA: 0x000020F1 File Offset: 0x000002F1
		public int EndLineNumber { get; set; }

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000007 RID: 7 RVA: 0x000020FA File Offset: 0x000002FA
		// (set) Token: 0x06000008 RID: 8 RVA: 0x00002102 File Offset: 0x00000302
		public string Name { get; set; }
	}
}
