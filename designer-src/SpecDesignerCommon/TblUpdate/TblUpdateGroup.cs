using System;
using System.Collections.Generic;
using System.Xml.Linq;

namespace SpecDesignerCommon.TblUpdate
{
	// Token: 0x02000132 RID: 306
	public class TblUpdateGroup : IComparable<TblUpdateGroup>
	{
		// Token: 0x170002DB RID: 731
		// (get) Token: 0x06000ADB RID: 2779 RVA: 0x000353F3 File Offset: 0x000335F3
		public List<TblUpdateItem> Items
		{
			get
			{
				return this._items;
			}
		}

		// Token: 0x170002DC RID: 732
		// (get) Token: 0x06000ADC RID: 2780 RVA: 0x000353FB File Offset: 0x000335FB
		// (set) Token: 0x06000ADD RID: 2781 RVA: 0x00035403 File Offset: 0x00033603
		public DateTime Date { get; set; }

		// Token: 0x06000ADE RID: 2782 RVA: 0x0003540C File Offset: 0x0003360C
		public TblUpdateGroup(XElement source)
		{
			this._source = source;
			this._items.Clear();
			if (this._source != null)
			{
				string value = this._source.Attribute("date").Value;
				this.Date = DateTime.Parse(value);
				foreach (XElement xelement in this._source.Elements())
				{
					this._items.Add(new TblUpdateItem(xelement));
				}
			}
		}

		// Token: 0x06000ADF RID: 2783 RVA: 0x000354BC File Offset: 0x000336BC
		public override string ToString()
		{
			if (this._source != null)
			{
				return this._source.ToString();
			}
			return null;
		}

		// Token: 0x06000AE0 RID: 2784 RVA: 0x000354D4 File Offset: 0x000336D4
		public int CompareTo(TblUpdateGroup other)
		{
			DateTime date = this.Date;
			DateTime date2 = other.Date;
			return this.Date.CompareTo(other.Date);
		}

		// Token: 0x0400040E RID: 1038
		private List<TblUpdateItem> _items = new List<TblUpdateItem>();

		// Token: 0x0400040F RID: 1039
		private XElement _source;
	}
}
