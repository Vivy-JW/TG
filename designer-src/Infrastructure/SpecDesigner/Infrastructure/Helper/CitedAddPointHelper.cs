using System;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000032 RID: 50
	public class CitedAddPointHelper
	{
		// Token: 0x0600014D RID: 333 RVA: 0x00006B54 File Offset: 0x00004D54
		public static string GetCitedContent(PackageKey program, string name)
		{
			if (SettingManager.Get().GetTzpManger(program).CitedAddPoint != null)
			{
				XElement xelement = (from XElement x in SettingManager.Get().GetTzpManger(program).CitedAddPoint.Elements("point")
					where x.Attribute("name").Value == name
					select x).ElementAtOrDefault<XElement>(0);
				if (xelement != null)
				{
					return xelement.Value;
				}
			}
			return null;
		}

		// Token: 0x0600014E RID: 334 RVA: 0x00006BF8 File Offset: 0x00004DF8
		public static bool HasCitedContent(PackageKey program, string name)
		{
			if (SettingManager.Get().GetTzpManger(program).CitedAddPoint != null)
			{
				XElement xelement = (from XElement x in SettingManager.Get().GetTzpManger(program).CitedAddPoint.Elements("point")
					where x.Attribute("name").Value == name
					select x).ElementAtOrDefault<XElement>(0);
				if (xelement != null)
				{
					return true;
				}
			}
			return false;
		}
	}
}
