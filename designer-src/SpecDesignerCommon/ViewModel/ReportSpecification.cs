using System;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000074 RID: 116
	public class ReportSpecification
	{
		// Token: 0x1700012F RID: 303
		// (get) Token: 0x0600046E RID: 1134 RVA: 0x000141C0 File Offset: 0x000123C0
		public string Ver
		{
			get
			{
				if (this._source != null && this._source.Attribute("ver") != null)
				{
					return this._source.Attribute("ver").Value;
				}
				throw new ArgumentNullException("Ver");
			}
		}

		// Token: 0x17000130 RID: 304
		// (get) Token: 0x0600046F RID: 1135 RVA: 0x00014211 File Offset: 0x00012411
		// (set) Token: 0x06000470 RID: 1136 RVA: 0x00014219 File Offset: 0x00012419
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x06000471 RID: 1137 RVA: 0x00014222 File Offset: 0x00012422
		public ReportSpecification(XElement element)
		{
			this._source = element;
		}

		// Token: 0x17000131 RID: 305
		// (get) Token: 0x06000472 RID: 1138 RVA: 0x00014231 File Offset: 0x00012431
		// (set) Token: 0x06000473 RID: 1139 RVA: 0x00014250 File Offset: 0x00012450
		public string Content
		{
			get
			{
				return this._source.Element("all").Value;
			}
			set
			{
				if (string.Equals(value, this.Content))
				{
					return;
				}
				this._source.Element("all").ReplaceNodes(new XCData(value));
				this._source.Element("all").Attribute("status").Value = "u";
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<ReportSpecChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x06000474 RID: 1140 RVA: 0x000142D5 File Offset: 0x000124D5
		public XElement ToXElement()
		{
			return this._source;
		}

		// Token: 0x06000475 RID: 1141 RVA: 0x000142E0 File Offset: 0x000124E0
		public static ReportSpecification Parse(string xml, TzpManager tzpManager)
		{
			if (string.IsNullOrEmpty(xml))
			{
				return null;
			}
			XElement xelement = XElement.Parse(xml, LoadOptions.None);
			return new ReportSpecification(xelement)
			{
				ProgramKey = new PackageKey(tzpManager.ProgramName, tzpManager.Type)
			};
		}

		// Token: 0x040001BC RID: 444
		private XElement _source;
	}
}
