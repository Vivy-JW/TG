using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000059 RID: 89
	public class ProgramSelectionChangedEventArgs
	{
		// Token: 0x170000B3 RID: 179
		// (get) Token: 0x060002F6 RID: 758 RVA: 0x0000C964 File Offset: 0x0000AB64
		// (set) Token: 0x060002F7 RID: 759 RVA: 0x0000C96C File Offset: 0x0000AB6C
		public PackageKey OldProgramKey { get; set; }

		// Token: 0x170000B4 RID: 180
		// (get) Token: 0x060002F8 RID: 760 RVA: 0x0000C975 File Offset: 0x0000AB75
		// (set) Token: 0x060002F9 RID: 761 RVA: 0x0000C97D File Offset: 0x0000AB7D
		public PackageKey NewProgramKey { get; set; }
	}
}
