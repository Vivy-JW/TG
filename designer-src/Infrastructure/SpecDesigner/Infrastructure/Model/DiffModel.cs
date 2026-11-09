using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000046 RID: 70
	public class DiffModel
	{
		// Token: 0x170000BA RID: 186
		// (get) Token: 0x06000222 RID: 546 RVA: 0x0000ABC5 File Offset: 0x00008DC5
		// (set) Token: 0x06000223 RID: 547 RVA: 0x0000ABCD File Offset: 0x00008DCD
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x170000BB RID: 187
		// (get) Token: 0x06000224 RID: 548 RVA: 0x0000ABD6 File Offset: 0x00008DD6
		public List<ADPModel> Customs
		{
			get
			{
				return this._customs;
			}
		}

		// Token: 0x170000BC RID: 188
		// (get) Token: 0x06000225 RID: 549 RVA: 0x0000ABDE File Offset: 0x00008DDE
		public XElement Merge
		{
			get
			{
				return this._merge;
			}
		}

		// Token: 0x06000226 RID: 550 RVA: 0x0000ACC4 File Offset: 0x00008EC4
		public DiffModel(XElement source, XElement tap, PackageKey key)
		{
			if (source == null)
			{
				return;
			}
			if (this._customs == null)
			{
				this._customs = new List<ADPModel>();
			}
			this.ProgramKey = key;
			if (source.Element("target2") != null)
			{
				using (IEnumerator<XElement> enumerator = source.Element("target2").Elements("point").GetEnumerator())
				{
					XElement point;
					while (enumerator.MoveNext())
					{
						XElement xelement = enumerator.Current;
						point = xelement;
						ADPModel adpmodel = new ADPModel(point.Attribute("name").Value);
						XElement xelement2 = (from t in tap.Elements("point")
							where t.Name == point.Name
							select t).ElementAtOrDefault<XElement>(0);
						if (xelement2 != null)
						{
							adpmodel.Element = point;
							adpmodel.IsDiff = true;
							this._customs.Add(adpmodel);
						}
					}
					return;
				}
			}
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsStandard)
			{
				IEnumerable<XElement> enumerable = from t in tap.Elements("point")
					where "c" == t.Attribute("src").Value
					select t;
				XElement p2;
				foreach (XElement xelement3 in enumerable)
				{
					p2 = xelement3;
					ADPModel adpmodel2 = new ADPModel(p2.Attribute("name").Value);
					this._customs.Add(adpmodel2);
					if (source.Element("target") != null)
					{
						XElement xelement4 = (from target in source.Elements("target").Elements<XElement>("point")
							where target.Attribute("name").Value == p2.Attribute("name").Value
							select target).ElementAtOrDefault<XElement>(0);
						adpmodel2.IsDiff = xelement4 != null;
						adpmodel2.Element = xelement4;
					}
				}
				this._merge = source.Element("topstd");
				return;
			}
			IEnumerable<XElement> enumerable2 = from t in tap.Elements("point")
				where "N" == t.Attribute("cite_std").Value
				select t;
			XElement p;
			foreach (XElement xelement5 in enumerable2)
			{
				p = xelement5;
				ADPModel adpmodel3 = new ADPModel(p.Attribute("name").Value);
				this._customs.Add(adpmodel3);
				if (source.Element("target") != null)
				{
					XElement xelement6 = (from target in source.Element("target").Elements("point")
						where target.Attribute("name").Value == p.Attribute("name").Value
						select target).ElementAtOrDefault<XElement>(0);
					adpmodel3.IsDiff = xelement6 != null;
					adpmodel3.Element = xelement6;
				}
			}
		}

		// Token: 0x040000DA RID: 218
		private List<ADPModel> _customs;

		// Token: 0x040000DB RID: 219
		private XElement _merge;
	}
}
