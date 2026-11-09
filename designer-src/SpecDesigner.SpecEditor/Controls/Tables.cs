using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001B RID: 27
	public class Tables
	{
		// Token: 0x060000BB RID: 187 RVA: 0x0000815F File Offset: 0x0000635F
		public Tables(XElement xml)
		{
			this._xml = xml;
			this._tables = TableColumnHelper.GetTables();
		}

		// Token: 0x060000BC RID: 188 RVA: 0x00008182 File Offset: 0x00006382
		public List<XElement> GetTables()
		{
			return this._tables.Where<XElement>((XElement t) => this.tableFilter(t)).ToList<XElement>();
		}

		// Token: 0x060000BD RID: 189 RVA: 0x00008208 File Offset: 0x00006408
		private bool tableFilter(XElement t)
		{
			IEnumerable<XElement> enumerable = from _t in this._xml.Elements("tbl")
				where _t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
				select _t;
			return enumerable.Where<XElement>((XElement _t) => t.Attribute("name").Value == _t.Attribute("name").Value).Count<XElement>() == 0;
		}

		// Token: 0x0400007E RID: 126
		private IEnumerable<XElement> _tables;

		// Token: 0x0400007F RID: 127
		private XElement _xml;
	}
}
