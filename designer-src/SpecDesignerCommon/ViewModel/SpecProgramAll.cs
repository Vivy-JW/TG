using System;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000015 RID: 21
	public class SpecProgramAll : SpecProgram
	{
		// Token: 0x060000A6 RID: 166 RVA: 0x00004411 File Offset: 0x00002611
		public SpecProgramAll(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x17000021 RID: 33
		// (get) Token: 0x060000A7 RID: 167 RVA: 0x0000441C File Offset: 0x0000261C
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
				return citeSTD.Elements("all").FirstOrDefault<XElement>();
			}
		}

		// Token: 0x060000A8 RID: 168 RVA: 0x00004478 File Offset: 0x00002678
		internal static SpecProgramAll Create(SpecificationInfo info)
		{
			XElement xelement = new XElement("all", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)),
				new XCData("")
			});
			return new SpecProgramAll(info.Key, xelement);
		}
	}
}
