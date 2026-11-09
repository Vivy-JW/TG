using System;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A7 RID: 167
	public class SpecActStringNode : AbstractStringNode
	{
		// Token: 0x06000718 RID: 1816 RVA: 0x0001FB56 File Offset: 0x0001DD56
		public SpecActStringNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x06000719 RID: 1817 RVA: 0x0001FB60 File Offset: 0x0001DD60
		internal static SpecActStringNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("sact", new object[]
			{
				new XAttribute("name", name),
				new XAttribute("text", ""),
				new XAttribute("lstr", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecActStringNode(info.Key, xelement);
		}

		// Token: 0x0600071A RID: 1818 RVA: 0x0001FBD8 File Offset: 0x0001DDD8
		internal void SetName(string newName)
		{
			base.SetAttribute("name", newName);
			base.OnPropertyChanged("Name");
		}
	}
}
