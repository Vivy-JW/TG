using System;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using System.Xml.Linq;

namespace SpecDesignerCommon.Site
{
	// Token: 0x02000142 RID: 322
	public class SiteFileHelper
	{
		// Token: 0x06000B36 RID: 2870 RVA: 0x00036264 File Offset: 0x00034464
		static SiteFileHelper()
		{
			using (RNGCryptoServiceProvider rngcryptoServiceProvider = new RNGCryptoServiceProvider())
			{
				rngcryptoServiceProvider.GetBytes(SiteFileHelper.entropy);
			}
		}

		// Token: 0x06000B37 RID: 2871 RVA: 0x000362BC File Offset: 0x000344BC
		public static string Open()
		{
			string siteFileText = SiteFileHelper.GetSiteFileText();
			return SiteFileHelper.Decrypt(siteFileText);
		}

		// Token: 0x06000B38 RID: 2872 RVA: 0x000362D8 File Offset: 0x000344D8
		public static void Save(XElement xml)
		{
			if (File.Exists("C:\\TT\\debug"))
			{
				xml.Save(Path.Combine(SiteFileHelper.GetSettingDir(), "sdsettings.xml"));
			}
			string text = SiteFileHelper.Encrypt(xml.ToString());
			File.WriteAllText(SiteFileHelper.FullPath(), text);
		}

		// Token: 0x06000B39 RID: 2873 RVA: 0x00036320 File Offset: 0x00034520
		public static void SaveTo(XElement xml, string path)
		{
			if (path == null)
			{
				SiteFileHelper.Save(xml);
			}
			string text = SiteFileHelper.Encrypt(xml.ToString());
			File.WriteAllText(path, text);
		}

		// Token: 0x06000B3A RID: 2874 RVA: 0x0003634C File Offset: 0x0003454C
		public static string Encrypt(string text)
		{
			byte[] bytes = Encoding.UTF8.GetBytes(text);
			byte[] bytes2 = Encoding.UTF8.GetBytes("T100 SpecDesigner");
			Rfc2898DeriveBytes rfc2898DeriveBytes = new Rfc2898DeriveBytes("T100 SiteManager.SiteFile", bytes2);
			AesManaged aesManaged = new AesManaged();
			aesManaged.BlockSize = aesManaged.LegalBlockSizes[0].MaxSize;
			aesManaged.KeySize = aesManaged.LegalKeySizes[0].MaxSize;
			aesManaged.Key = rfc2898DeriveBytes.GetBytes(aesManaged.KeySize / 8);
			aesManaged.IV = rfc2898DeriveBytes.GetBytes(aesManaged.BlockSize / 8);
			byte[] array = null;
			using (ICryptoTransform cryptoTransform = aesManaged.CreateEncryptor())
			{
				using (MemoryStream memoryStream = new MemoryStream())
				{
					using (CryptoStream cryptoStream = new CryptoStream(memoryStream, cryptoTransform, CryptoStreamMode.Write))
					{
						cryptoStream.Write(bytes, 0, bytes.Length);
						cryptoStream.FlushFinalBlock();
						array = memoryStream.ToArray();
					}
				}
			}
			return Convert.ToBase64String(array);
		}

		// Token: 0x06000B3B RID: 2875 RVA: 0x00036468 File Offset: 0x00034668
		public static string Decrypt(string text)
		{
			byte[] array = Convert.FromBase64String(text);
			byte[] bytes = Encoding.UTF8.GetBytes("T100 SpecDesigner");
			Rfc2898DeriveBytes rfc2898DeriveBytes = new Rfc2898DeriveBytes("T100 SiteManager.SiteFile", bytes);
			AesManaged aesManaged = new AesManaged();
			aesManaged.BlockSize = aesManaged.LegalBlockSizes[0].MaxSize;
			aesManaged.KeySize = aesManaged.LegalKeySizes[0].MaxSize;
			aesManaged.Key = rfc2898DeriveBytes.GetBytes(aesManaged.KeySize / 8);
			aesManaged.IV = rfc2898DeriveBytes.GetBytes(aesManaged.BlockSize / 8);
			string text2 = null;
			using (ICryptoTransform cryptoTransform = aesManaged.CreateDecryptor())
			{
				using (MemoryStream memoryStream = new MemoryStream(array, 0, array.Length))
				{
					using (CryptoStream cryptoStream = new CryptoStream(memoryStream, cryptoTransform, CryptoStreamMode.Read))
					{
						try
						{
							using (StreamReader streamReader = new StreamReader(cryptoStream))
							{
								text2 = streamReader.ReadToEnd();
							}
						}
						catch (Exception)
						{
						}
					}
				}
			}
			return text2;
		}

		// Token: 0x06000B3C RID: 2876 RVA: 0x000365A0 File Offset: 0x000347A0
		private static string GetSiteFileText()
		{
			string settingDir = SiteFileHelper.GetSettingDir();
			string text = Path.Combine(settingDir, SiteFileHelper.SiteFileName);
			if (File.Exists("C:\\TT\\debug"))
			{
				string text2 = Path.Combine(settingDir, "sdsettings.xml");
				if (File.Exists(text2))
				{
					try
					{
						return SiteFileHelper.Encrypt(File.ReadAllText(text2));
					}
					catch
					{
					}
				}
			}
			if (!File.Exists(text))
			{
				return SiteFileHelper.Encrypt("<Settings />");
			}
			return File.ReadAllText(text);
		}

		// Token: 0x06000B3D RID: 2877 RVA: 0x0003661C File Offset: 0x0003481C
		public static string GetSettingDir()
		{
			string text = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData, Environment.SpecialFolderOption.Create);
			text = Path.Combine(text, "SpecDesigner");
			if (!Directory.Exists(text))
			{
				Directory.CreateDirectory(text);
			}
			return text;
		}

		// Token: 0x06000B3E RID: 2878 RVA: 0x00036652 File Offset: 0x00034852
		public static string FullPath()
		{
			return Path.Combine(SiteFileHelper.GetSettingDir(), SiteFileHelper.SiteFileName);
		}

		// Token: 0x04000452 RID: 1106
		private static byte[] entropy = new byte[25];

		// Token: 0x04000453 RID: 1107
		private static string SiteFileName = "sites.xml";
	}
}
