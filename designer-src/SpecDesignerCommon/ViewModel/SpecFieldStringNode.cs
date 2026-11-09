using System;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000091 RID: 145
	public class SpecFieldStringNode : AbstractStringNode
	{
		// Token: 0x060005EA RID: 1514 RVA: 0x0001B113 File Offset: 0x00019313
		public SpecFieldStringNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x060005EB RID: 1515 RVA: 0x0001B120 File Offset: 0x00019320
		internal static SpecFieldStringNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("sfield", new object[]
			{
				new XAttribute("name", name),
				new XAttribute("text", ""),
				new XAttribute("lstr", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecFieldStringNode(info.Key, xelement);
		}
	}
}
