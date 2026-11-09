using System;
using System.IO;
using System.Xml.Linq;

namespace SpecDesignerCommon
{
	// Token: 0x020000EA RID: 234
	public class SettingFileHelper
	{
		// Token: 0x17000229 RID: 553
		// (get) Token: 0x060007D9 RID: 2009 RVA: 0x0002307E File Offset: 0x0002127E
		private static string FullPath
		{
			get
			{
				return Path.Combine(SettingFileHelper.GetSettingDir(), "Preference.xml");
			}
		}

		// Token: 0x1700022A RID: 554
		// (get) Token: 0x060007DA RID: 2010 RVA: 0x0002308F File Offset: 0x0002128F
		public static string FullPathForLayoutSetting
		{
			get
			{
				return Path.Combine(SettingFileHelper.GetSettingDir(), "LayoutSetting.xml");
			}
		}

		// Token: 0x060007DC RID: 2012 RVA: 0x000230A8 File Offset: 0x000212A8
		public static string Open()
		{
			return SettingFileHelper.GetFileText();
		}

		// Token: 0x060007DD RID: 2013 RVA: 0x000230B0 File Offset: 0x000212B0
		private static string GetFileText()
		{
			string fullPath = SettingFileHelper.FullPath;
			if (!File.Exists(fullPath))
			{
				return "<Settings />";
			}
			string text = File.ReadAllText(fullPath);
			try
			{
				XElement.Parse(text);
			}
			catch
			{
				return "<Settings />";
			}
			return text;
		}

		// Token: 0x060007DE RID: 2014 RVA: 0x00023100 File Offset: 0x00021300
		private static string GetSettingDir()
		{
			string text = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData, Environment.SpecialFolderOption.Create);
			text = Path.Combine(text, "SpecDesigner");
			if (!Directory.Exists(text))
			{
				Directory.CreateDirectory(text);
			}
			return text;
		}

		// Token: 0x060007DF RID: 2015 RVA: 0x00023136 File Offset: 0x00021336
		public static void Save(XElement xml)
		{
			File.WriteAllText(SettingFileHelper.FullPath, xml.ToString());
		}
	}
}
