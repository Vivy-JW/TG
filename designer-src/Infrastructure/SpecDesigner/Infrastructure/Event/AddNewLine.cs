using System;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x02000017 RID: 23
	public class AddNewLine
	{
		// Token: 0x17000015 RID: 21
		// (get) Token: 0x06000043 RID: 67 RVA: 0x0000239E File Offset: 0x0000059E
		// (set) Token: 0x06000044 RID: 68 RVA: 0x000023A6 File Offset: 0x000005A6
		public int LineNumber { get; set; }

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x06000045 RID: 69 RVA: 0x000023AF File Offset: 0x000005AF
		// (set) Token: 0x06000046 RID: 70 RVA: 0x000023B7 File Offset: 0x000005B7
		public PackageKey ProgramKey { get; set; }
	}
}
