using System;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000054 RID: 84
	public class SpecProgramMI : SpecProgram
	{
		// Token: 0x060002CE RID: 718 RVA: 0x0000C245 File Offset: 0x0000A445
		public SpecProgramMI(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x170000AC RID: 172
		// (get) Token: 0x060002CF RID: 719 RVA: 0x0000C250 File Offset: 0x0000A450
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
				return citeSTD.Elements("mi_all").FirstOrDefault<XElement>();
			}
		}

		// Token: 0x060002D0 RID: 720 RVA: 0x0000C2AC File Offset: 0x0000A4AC
		internal static SpecProgramMI Create(SpecificationInfo info)
		{
			XElement xelement = new XElement("mi_all", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)),
				new XCData("")
			});
			return new SpecProgramMI(info.Key, xelement);
		}
	}
}
