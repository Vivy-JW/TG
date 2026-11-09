using System;
using Microsoft.Win32;

namespace SpecDesignerCommon.Site
{
	// Token: 0x02000103 RID: 259
	public class RegistryReader
	{
		// Token: 0x06000917 RID: 2327 RVA: 0x0002BFAC File Offset: 0x0002A1AC
		private static string GetInstallDir(string version)
		{
			string text = string.Format("SOFTWARE\\{0}\\{1}_{2}", "DSC Development Tools", "T100Designer", version);
			RegistryKey registryKey = RegistryKey.OpenBaseKey(RegistryHive.LocalMachine, RegistryView.Registry32);
			RegistryKey registryKey2 = registryKey.OpenSubKey(text, true);
			if (registryKey2 == null)
			{
				return null;
			}
			return registryKey2.GetValue("InstallDir") as string;
		}

		// Token: 0x06000918 RID: 2328 RVA: 0x0002C000 File Offset: 0x0002A200
		public static string GetExeDir(string version)
		{
			string installDir = RegistryReader.GetInstallDir(version);
			if (string.IsNullOrWhiteSpace(installDir))
			{
				throw new MissingFieldException();
			}
			return string.Format("{0}{1}", installDir, "T100Designer.exe");
		}
	}
}
