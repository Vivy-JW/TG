using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x0200008D RID: 141
	public class CodeLibrariesHelper
	{
		// Token: 0x170001AC RID: 428
		// (get) Token: 0x060005C6 RID: 1478 RVA: 0x0001A635 File Offset: 0x00018835
		public static CodeLibrariesHelper This
		{
			get
			{
				if (CodeLibrariesHelper._this == null)
				{
					CodeLibrariesHelper._this = new CodeLibrariesHelper();
					CodeLibrariesHelper.This.Parse(SettingManager.Get().Info_Subroutines);
					CodeLibrariesHelper.This.Parse(SettingManager.Get().Info_Libraries);
				}
				return CodeLibrariesHelper._this;
			}
		}

		// Token: 0x060005C7 RID: 1479 RVA: 0x0001A678 File Offset: 0x00018878
		private void Parse(string content)
		{
			foreach (XElement xelement in CodeLibrariesHelper.libraries)
			{
				if (xelement.ToString() == content)
				{
					return;
				}
			}
			XElement xelement2 = XElement.Parse(content);
			CodeLibrariesHelper.libraries.Add(xelement2);
		}

		// Token: 0x060005C8 RID: 1480 RVA: 0x0001A924 File Offset: 0x00018B24
		public IEnumerable<string> GetLibraries()
		{
			foreach (XElement lib in CodeLibrariesHelper.libraries)
			{
				foreach (XElement com in lib.Elements("com"))
				{
					yield return com.Attribute("id").Value;
				}
			}
			yield break;
		}

		// Token: 0x060005C9 RID: 1481 RVA: 0x0001A96C File Offset: 0x00018B6C
		public string GetDescription(string name)
		{
			if (!string.IsNullOrWhiteSpace(name))
			{
				foreach (XElement xelement in CodeLibrariesHelper.libraries)
				{
					XElement xelement2 = (from element in xelement.Elements("com")
						where element.Attribute("id").Value == name
						select element).ElementAtOrDefault<XElement>(0);
					if (xelement2 != null)
					{
						return xelement2.Attribute("desc").Value;
					}
				}
			}
			return null;
		}

		// Token: 0x04000235 RID: 565
		private static CodeLibrariesHelper _this = null;

		// Token: 0x04000236 RID: 566
		private static List<XElement> libraries = new List<XElement>();
	}
}
