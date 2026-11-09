using System;

namespace SpecDesignerCommon
{
	// Token: 0x020000E2 RID: 226
	public class SearchFromFocusEventArgs
	{
		// Token: 0x060007A0 RID: 1952 RVA: 0x000223BC File Offset: 0x000205BC
		public SearchFromFocusEventArgs(PackageKey key, SearchDirection direction)
		{
			this.ProgramKey = key;
			this.Direction = direction;
		}

		// Token: 0x17000220 RID: 544
		// (get) Token: 0x060007A1 RID: 1953 RVA: 0x000223D2 File Offset: 0x000205D2
		// (set) Token: 0x060007A2 RID: 1954 RVA: 0x000223DA File Offset: 0x000205DA
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000221 RID: 545
		// (get) Token: 0x060007A3 RID: 1955 RVA: 0x000223E3 File Offset: 0x000205E3
		// (set) Token: 0x060007A4 RID: 1956 RVA: 0x000223EB File Offset: 0x000205EB
		public SearchDirection Direction { get; set; }
	}
}
