using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesignerPreference
{
	// Token: 0x02000089 RID: 137
	public sealed class LanguageModel
	{
		// Token: 0x170001A5 RID: 421
		// (get) Token: 0x060005AE RID: 1454 RVA: 0x0001A3B2 File Offset: 0x000185B2
		// (set) Token: 0x060005AF RID: 1455 RVA: 0x0001A3BA File Offset: 0x000185BA
		public string Lang { get; set; }

		// Token: 0x170001A6 RID: 422
		// (get) Token: 0x060005B0 RID: 1456 RVA: 0x0001A3C3 File Offset: 0x000185C3
		// (set) Token: 0x060005B1 RID: 1457 RVA: 0x0001A3CB File Offset: 0x000185CB
		public string UIID { get; set; }

		// Token: 0x170001A7 RID: 423
		// (get) Token: 0x060005B2 RID: 1458 RVA: 0x0001A3D4 File Offset: 0x000185D4
		// (set) Token: 0x060005B3 RID: 1459 RVA: 0x0001A3DC File Offset: 0x000185DC
		public string DataID { get; set; }

		// Token: 0x170001A8 RID: 424
		// (get) Token: 0x060005B4 RID: 1460 RVA: 0x0001A3E5 File Offset: 0x000185E5
		// (set) Token: 0x060005B5 RID: 1461 RVA: 0x0001A3ED File Offset: 0x000185ED
		public string Name { get; set; }

		// Token: 0x060005B6 RID: 1462 RVA: 0x0001A3F6 File Offset: 0x000185F6
		public LanguageModel()
		{
		}

		// Token: 0x060005B7 RID: 1463 RVA: 0x0001A3FE File Offset: 0x000185FE
		public LanguageModel(string uiID, string dataID, string name)
		{
			this.UIID = uiID;
			this.DataID = dataID;
			this.Name = name;
			this.Lang = uiID.Replace('_', '-');
		}

		// Token: 0x060005B8 RID: 1464 RVA: 0x0001A480 File Offset: 0x00018680
		public static IEnumerable<LanguageModel> Load()
		{
			string info_Languages = SettingManager.Get().Info_Languages;
			IEnumerable<LanguageModel> enumerable = null;
			if (!string.IsNullOrEmpty(info_Languages))
			{
				XElement xelement = XElement.Parse(info_Languages);
				enumerable = from el in xelement.Elements("language")
					select new LanguageModel((string)el.Attribute("ui_id"), (string)el.Attribute("data_id"), (string)el.Attribute("desc"));
			}
			return enumerable;
		}
	}
}
