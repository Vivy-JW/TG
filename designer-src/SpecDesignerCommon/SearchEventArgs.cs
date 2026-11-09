using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000135 RID: 309
	public class SearchEventArgs : EventArgs
	{
		// Token: 0x170002DD RID: 733
		// (get) Token: 0x06000AF4 RID: 2804 RVA: 0x00035503 File Offset: 0x00033703
		// (set) Token: 0x06000AF5 RID: 2805 RVA: 0x0003550B File Offset: 0x0003370B
		public SearchTarget Type { get; private set; }

		// Token: 0x06000AF6 RID: 2806 RVA: 0x00035514 File Offset: 0x00033714
		public SearchEventArgs(SearchTarget type)
		{
			this.Type = type;
		}
	}
}
