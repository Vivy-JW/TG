using System;
using System.Collections.Generic;
using System.Linq;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000149 RID: 329
	public class MultiSelectionArgs
	{
		// Token: 0x170002FD RID: 765
		// (get) Token: 0x06000B8D RID: 2957 RVA: 0x00038C7D File Offset: 0x00036E7D
		// (set) Token: 0x06000B8E RID: 2958 RVA: 0x00038C85 File Offset: 0x00036E85
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x170002FE RID: 766
		// (get) Token: 0x06000B8F RID: 2959 RVA: 0x00038C8E File Offset: 0x00036E8E
		// (set) Token: 0x06000B90 RID: 2960 RVA: 0x00038C96 File Offset: 0x00036E96
		public List<string> Components { get; private set; }

		// Token: 0x06000B91 RID: 2961 RVA: 0x00038C9F File Offset: 0x00036E9F
		public MultiSelectionArgs(PackageKey key, IEnumerable<string> components)
		{
			this.ProgramKey = key;
			this.Components = components.ToList<string>();
		}
	}
}
