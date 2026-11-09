using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000022 RID: 34
	public class NormalizationSearchResult
	{
		// Token: 0x17000039 RID: 57
		// (get) Token: 0x06000106 RID: 262 RVA: 0x00005E89 File Offset: 0x00004089
		// (set) Token: 0x06000107 RID: 263 RVA: 0x00005E91 File Offset: 0x00004091
		public int StartOffset { get; set; }

		// Token: 0x1700003A RID: 58
		// (get) Token: 0x06000108 RID: 264 RVA: 0x00005E9A File Offset: 0x0000409A
		// (set) Token: 0x06000109 RID: 265 RVA: 0x00005EA2 File Offset: 0x000040A2
		public int Length { get; set; }

		// Token: 0x1700003B RID: 59
		// (get) Token: 0x0600010A RID: 266 RVA: 0x00005EAB File Offset: 0x000040AB
		// (set) Token: 0x0600010B RID: 267 RVA: 0x00005EB3 File Offset: 0x000040B3
		public int LineNumber { get; set; }
	}
}
