using System;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon
{
	// Token: 0x020000B0 RID: 176
	public class SpecArgs
	{
		// Token: 0x1700021D RID: 541
		// (get) Token: 0x06000765 RID: 1893 RVA: 0x000210C5 File Offset: 0x0001F2C5
		// (set) Token: 0x06000766 RID: 1894 RVA: 0x000210CD File Offset: 0x0001F2CD
		public string Name { get; set; }

		// Token: 0x1700021E RID: 542
		// (get) Token: 0x06000767 RID: 1895 RVA: 0x000210D6 File Offset: 0x0001F2D6
		// (set) Token: 0x06000768 RID: 1896 RVA: 0x000210DE File Offset: 0x0001F2DE
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x1700021F RID: 543
		// (get) Token: 0x06000769 RID: 1897 RVA: 0x000210E7 File Offset: 0x0001F2E7
		// (set) Token: 0x0600076A RID: 1898 RVA: 0x000210EF File Offset: 0x0001F2EF
		public ComponentType Type { get; private set; }

		// Token: 0x0600076B RID: 1899 RVA: 0x000210F8 File Offset: 0x0001F2F8
		public SpecArgs(string name, ComponentType type, PackageKey key)
		{
			this.Name = name;
			this.ProgramKey = key;
			this.Type = type;
		}
	}
}
