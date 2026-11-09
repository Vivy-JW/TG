using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000078 RID: 120
	public interface ISearchResult
	{
		// Token: 0x1700013C RID: 316
		// (get) Token: 0x06000496 RID: 1174
		// (set) Token: 0x06000497 RID: 1175
		string Locator { get; set; }

		// Token: 0x1700013D RID: 317
		// (get) Token: 0x06000498 RID: 1176
		// (set) Token: 0x06000499 RID: 1177
		string Content { get; set; }
	}
}
