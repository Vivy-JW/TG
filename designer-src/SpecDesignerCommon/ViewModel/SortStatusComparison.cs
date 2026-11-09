using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000041 RID: 65
	internal class SortStatusComparison : IComparer<AbstractSpecNode>
	{
		// Token: 0x0600021D RID: 541 RVA: 0x00009600 File Offset: 0x00007800
		public int Compare(AbstractSpecNode a, AbstractSpecNode b)
		{
			if (a == null)
			{
				if (b != null)
				{
					return -1;
				}
				return 0;
			}
			else
			{
				if (b == null)
				{
					return 1;
				}
				return 0;
			}
		}
	}
}
