using System;
using System.Collections.Generic;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000067 RID: 103
	public class TabIndexSortHelper
	{
		// Token: 0x060003F8 RID: 1016 RVA: 0x000129C3 File Offset: 0x00010BC3
		public void SetAsNonTabable(XmlElement m)
		{
			m["tabIndex"] = "";
			this._sourceList.Add(m);
		}

		// Token: 0x060003F9 RID: 1017 RVA: 0x000129E4 File Offset: 0x00010BE4
		private void SortSourceList(bool autoIndex)
		{
			if (autoIndex)
			{
				int num = 1;
				foreach (XmlElement xmlElement in this._sourceList)
				{
					xmlElement["tabIndex"] = num.ToString();
					num++;
				}
			}
		}

		// Token: 0x04000185 RID: 389
		private List<XmlElement> _sourceList = new List<XmlElement>();

		// Token: 0x04000186 RID: 390
		private List<XmlElement> _tabIndexedList = new List<XmlElement>();
	}
}
