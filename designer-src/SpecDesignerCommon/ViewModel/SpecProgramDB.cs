using System;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000028 RID: 40
	public class SpecProgramDB : SpecProgram
	{
		// Token: 0x06000155 RID: 341 RVA: 0x00007134 File Offset: 0x00005334
		public SpecProgramDB(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x17000053 RID: 83
		// (get) Token: 0x06000156 RID: 342 RVA: 0x00007140 File Offset: 0x00005340
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
				return citeSTD.Elements("db_all").FirstOrDefault<XElement>();
			}
		}

		// Token: 0x06000157 RID: 343 RVA: 0x0000719C File Offset: 0x0000539C
		internal static SpecProgramDB Create(SpecificationInfo info)
		{
			XElement xelement = new XElement("db_all", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)),
				new XCData("")
			});
			return new SpecProgramDB(info.Key, xelement);
		}
	}
}
