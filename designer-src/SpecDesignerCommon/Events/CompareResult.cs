using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000023 RID: 35
	public class CompareResult : IComparer<NormalizationSearchResult>
	{
		// Token: 0x0600010C RID: 268 RVA: 0x00005EBC File Offset: 0x000040BC
		public int Compare(NormalizationSearchResult x, NormalizationSearchResult y)
		{
			if (x.StartOffset == y.StartOffset && x.Length == y.Length && x.LineNumber == y.LineNumber)
			{
				return 0;
			}
			return -1;
		}
	}
}
