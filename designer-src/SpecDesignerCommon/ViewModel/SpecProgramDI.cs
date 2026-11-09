using System;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000F1 RID: 241
	public class SpecProgramDI : SpecProgram
	{
		// Token: 0x06000806 RID: 2054 RVA: 0x0002386D File Offset: 0x00021A6D
		public SpecProgramDI(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x17000233 RID: 563
		// (get) Token: 0x06000807 RID: 2055 RVA: 0x00023878 File Offset: 0x00021A78
		public override XElement CitedSpec
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.ProgramKey).IsStandardProgram)
				{
					return null;
				}
				XElement citeSTD = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.CiteSTD;
				if (citeSTD == null)
				{
					return null;
				}
				return citeSTD.Elements("di_all").FirstOrDefault<XElement>();
			}
		}

		// Token: 0x06000808 RID: 2056 RVA: 0x000238D4 File Offset: 0x00021AD4
		internal static SpecProgramDI Create(SpecificationInfo info)
		{
			XElement xelement = new XElement("di_all", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)),
				new XCData("")
			});
			return new SpecProgramDI(info.Key, xelement);
		}
	}
}
