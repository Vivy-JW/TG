using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000113 RID: 275
	public class TzpSavedEventArgs : EventArgs
	{
		// Token: 0x1700029D RID: 669
		// (get) Token: 0x060009D8 RID: 2520 RVA: 0x000313EC File Offset: 0x0002F5EC
		// (set) Token: 0x060009D9 RID: 2521 RVA: 0x000313F4 File Offset: 0x0002F5F4
		public TzpType Type { get; private set; }

		// Token: 0x1700029E RID: 670
		// (get) Token: 0x060009DA RID: 2522 RVA: 0x000313FD File Offset: 0x0002F5FD
		// (set) Token: 0x060009DB RID: 2523 RVA: 0x00031405 File Offset: 0x0002F605
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x060009DC RID: 2524 RVA: 0x0003140E File Offset: 0x0002F60E
		public TzpSavedEventArgs(PackageKey key, TzpType type)
		{
			this.Type = type;
			this.ProgramKey = key;
		}
	}
}
