using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000061 RID: 97
	public class SortPosition : IComparer<XmlElement>
	{
		// Token: 0x060003B3 RID: 947 RVA: 0x00010DBC File Offset: 0x0000EFBC
		public int Compare(XmlElement x, XmlElement y)
		{
			int num = x.GridX - y.GridX;
			if (num == 0)
			{
				return x.GridY - y.GridY;
			}
			return num;
		}
	}
}
