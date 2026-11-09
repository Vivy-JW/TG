using System;
using System.Collections.Generic;
using System.Xml.Linq;

namespace SpecDesignerCommon.TblUpdate
{
	// Token: 0x02000072 RID: 114
	public class TblUpdateCastItem
	{
		// Token: 0x1700012C RID: 300
		// (get) Token: 0x06000466 RID: 1126 RVA: 0x000140C5 File Offset: 0x000122C5
		public List<TblUpdateGroup> Items
		{
			get
			{
				return this._items;
			}
		}

		// Token: 0x06000467 RID: 1127 RVA: 0x000140CD File Offset: 0x000122CD
		public TblUpdateCastItem()
		{
		}

		// Token: 0x06000468 RID: 1128 RVA: 0x000140D8 File Offset: 0x000122D8
		public TblUpdateCastItem(XElement source)
			: this()
		{
			this._source = source;
			this._items = new List<TblUpdateGroup>();
			if (this._source != null && this._source.HasElements)
			{
				foreach (XElement xelement in this._source.Elements())
				{
					TblUpdateGroup tblUpdateGroup = new TblUpdateGroup(xelement);
					if (tblUpdateGroup != null)
					{
						this._items.Add(tblUpdateGroup);
					}
				}
			}
		}

		// Token: 0x06000469 RID: 1129 RVA: 0x00014168 File Offset: 0x00012368
		public static TblUpdateCastItem Parse(string content)
		{
			XElement xelement = XElement.Parse(content);
			return new TblUpdateCastItem(xelement);
		}

		// Token: 0x0600046A RID: 1130 RVA: 0x00014182 File Offset: 0x00012382
		public override string ToString()
		{
			if (this._source != null)
			{
				return this._source.ToString();
			}
			return null;
		}

		// Token: 0x040001B8 RID: 440
		private List<TblUpdateGroup> _items;

		// Token: 0x040001B9 RID: 441
		private XElement _source;
	}
}
