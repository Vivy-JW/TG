using System;
using System.Collections.Generic;
using System.IO;
using System.Security.AccessControl;
using System.Security.Principal;
using System.Text;
using System.Windows;
using ICSharpCode.SharpZipLib.Core;
using ICSharpCode.SharpZipLib.Zip;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon
{
	// Token: 0x020000B2 RID: 178
	public class PackageManager
	{
		// Token: 0x0600076C RID: 1900 RVA: 0x00021118 File Offset: 0x0001F318
		public static void UnpackingCited(string zipFilePath, TzpManager tzpManager)
		{
			if (!File.Exists(zipFilePath))
			{
				string text = string.Format((string)Application.Current.FindResource("Message_NoCitedProg"), zipFilePath);
				throw new FileNotFoundException(text);
			}
			using (ZipInputStream zipInputStream = new ZipInputStream(File.OpenRead(zipFilePath)))
			{
				zipInputStream.IsStreamOwner = true;
				string text2 = string.Empty;
				ZipEntry nextEntry;
				while ((nextEntry = zipInputStream.GetNextEntry()) != null)
				{
					if (nextEntry.IsFile)
					{
						text2 = nextEntry.Name;
						StreamReader streamReader = new StreamReader(zipInputStream);
						string text3 = streamReader.ReadToEnd();
						new FileInfo(text2);
						string extension;
						if ((extension = Path.GetExtension(text2)) != null)
						{
							if (!(extension == ".tap"))
							{
								if (!(extension == ".4fd"))
								{
									if (!(extension == ".tsd"))
									{
										if (!(extension == ".4ad") && !(extension == ".csd"))
										{
										}
									}
									else
									{
										tzpManager.LoadCiteTsd(text3);
									}
								}
							}
							else
							{
								tzpManager.LoadCitedAddPoint(text3);
							}
						}
					}
				}
			}
		}

		// Token: 0x0600076D RID: 1901 RVA: 0x00021228 File Offset: 0x0001F428
		public static void Unpacking(string zipFilePath, TzpManager tzpManager)
		{
			List<string> list = new List<string>();
			switch (tzpManager.Type)
			{
			case TzpType.Report:
				list.Add(".tap");
				list.Add(".tgl");
				break;
			case TzpType.ReportSpec:
				list.Add(".rsd");
				break;
			case TzpType.Code:
				list.Add(".tap");
				list.Add(".tgl");
				if (tzpManager.IsDiff)
				{
					list.Add(".src");
					list.Add(".apt");
				}
				break;
			case TzpType.CodeSpec:
				list.Add(".csd");
				break;
			case TzpType.Form:
				list.Add(".tsd");
				list.Add(".4fd");
				if (tzpManager.IsSimpleForm)
				{
					list.Add(".4fdref");
				}
				break;
			}
			tzpManager.LoadActionDefaults(SettingManager.Get().Info_ActionDefaults);
			using (ZipInputStream zipInputStream = new ZipInputStream(File.OpenRead(zipFilePath)))
			{
				zipInputStream.IsStreamOwner = true;
				string text = string.Empty;
				ZipEntry nextEntry;
				while ((nextEntry = zipInputStream.GetNextEntry()) != null)
				{
					if (nextEntry.IsFile)
					{
						text = nextEntry.Name;
						StreamReader streamReader = new StreamReader(zipInputStream);
						string text2 = streamReader.ReadToEnd();
						new FileInfo(text);
						string text3 = Path.GetExtension(text);
						if (list.Contains(text3))
						{
							list.Remove(text3);
						}
						string text4;
						switch (text4 = text3)
						{
						case ".tap":
							tzpManager.LoadAddPoint(text2);
							break;
						case ".tap2":
							tzpManager.LoadUpdatedAddPoint(text2);
							break;
						case ".tgl":
							tzpManager.LoadCodeFile(text2);
							break;
						case ".4gl":
							tzpManager.Load4glFile(text2);
							break;
						case ".4fd":
							tzpManager.LoadFormFile(text2);
							break;
						case ".4fdref":
							if (tzpManager.IsSimpleForm)
							{
								tzpManager.LoadRefFormFile(text2);
							}
							break;
						case ".tsd":
							tzpManager.LoadSpecificationInfo(text2);
							break;
						case ".tsd2":
							tzpManager.LoadUpdatedSpecificationInfo(text2);
							break;
						case ".4ad":
							tzpManager.LoadCustomActionDefaults(text2);
							break;
						case ".merge":
							tzpManager.LoadFormDiffList(text2);
							break;
						case ".csd":
							tzpManager.LoadSpecificationForCode(text2);
							break;
						case ".rsd":
							tzpManager.LoadSpecificationForReport(text2);
							break;
						case ".src":
						{
							text3 = Path.GetExtension(Path.GetFileNameWithoutExtension(text));
							string text5;
							if ((text5 = text3) != null)
							{
								if (!(text5 == ".4gl"))
								{
									if (text5 == ".tap")
									{
										tzpManager.LoadDiffSrcTAPFile(text2);
									}
								}
								else
								{
									tzpManager.LoadDiffSrcFile(text2);
								}
							}
							break;
						}
						case ".apt":
							tzpManager.LoadADT(text2);
							break;
						case ".bdx":
							tzpManager.LoadBindings(text2);
							break;
						case ".xml":
							tzpManager.LoadinfoXML(text2);
							break;
						}
					}
				}
				if (list.Count > 0)
				{
					throw new FileNotFoundException(string.Format((string)Application.Current.FindResource("Message_ZipFileNotEnough"), new object[]
					{
						Path.GetFileName(zipFilePath),
						Environment.NewLine,
						string.Join(Environment.NewLine, list.ToArray()),
						Environment.NewLine
					}));
				}
			}
		}

		// Token: 0x0600076E RID: 1902 RVA: 0x00021650 File Offset: 0x0001F850
		public static bool Packing(string zipfile, TzpManager tzpManager)
		{
			ZipInputStream zipInputStream = new ZipInputStream(File.OpenRead(zipfile));
			MemoryStream memoryStream = new MemoryStream();
			ZipOutputStream zipOutputStream = new ZipOutputStream(memoryStream);
			zipOutputStream.IsStreamOwner = false;
			zipOutputStream.SetLevel(3);
			string text = string.Empty;
			bool flag = false;
			ZipEntry nextEntry;
			while ((nextEntry = zipInputStream.GetNextEntry()) != null)
			{
				if (nextEntry.IsFile)
				{
					text = nextEntry.Name;
					string extension;
					if ((extension = Path.GetExtension(text)) == null)
					{
						goto IL_08DB;
					}
					if (<PrivateImplementationDetails>{9B9CE70C-F808-4287-A7F1-49081D8E35B4}.$$method0x6000726-1 == null)
					{
						<PrivateImplementationDetails>{9B9CE70C-F808-4287-A7F1-49081D8E35B4}.$$method0x6000726-1 = new Dictionary<string, int>(13)
						{
							{ ".rsd", 0 },
							{ ".csd", 1 },
							{ ".tap", 2 },
							{ ".tap2", 3 },
							{ ".src", 4 },
							{ ".tgl", 5 },
							{ ".4gl", 6 },
							{ ".str", 7 },
							{ ".4fd", 8 },
							{ ".tsd", 9 },
							{ ".tsd2", 10 },
							{ ".4ad", 11 },
							{ ".bdx", 12 }
						};
					}
					int num;
					if (!<PrivateImplementationDetails>{9B9CE70C-F808-4287-A7F1-49081D8E35B4}.$$method0x6000726-1.TryGetValue(extension, out num))
					{
						goto IL_08DB;
					}
					switch (num)
					{
					case 0:
					{
						string text2 = tzpManager.RSD;
						if (string.IsNullOrEmpty(text2))
						{
							goto IL_0935;
						}
						text2 = text2.Replace("><![CDATA[", ">\r\n<![CDATA[");
						text2 = text2.Replace("]]><", "]]>\r\n<");
						using (MemoryStream memoryStream2 = new MemoryStream(Encoding.UTF8.GetBytes(text2)))
						{
							zipOutputStream.PutNextEntry(new ZipEntry(text)
							{
								DateTime = DateTime.Now
							});
							StreamUtils.Copy(memoryStream2, zipOutputStream, new byte[4096]);
							zipOutputStream.CloseEntry();
							goto IL_0935;
						}
						break;
					}
					case 1:
						break;
					case 2:
						goto IL_028A;
					case 3:
						goto IL_0351;
					case 4:
						goto IL_03E3;
					case 5:
						goto IL_0527;
					case 6:
						goto IL_0593;
					case 7:
						goto IL_05EF;
					case 8:
						goto IL_065B;
					case 9:
						goto IL_06C7;
					case 10:
						goto IL_078D;
					case 11:
						goto IL_0813;
					case 12:
						goto IL_086F;
					default:
						goto IL_08DB;
					}
					string text3 = tzpManager.CSD;
					if (string.IsNullOrEmpty(text3))
					{
						goto IL_0935;
					}
					text3 = text3.Replace("><![CDATA[", ">\r\n<![CDATA[");
					text3 = text3.Replace("]]><", "]]>\r\n<");
					using (MemoryStream memoryStream3 = new MemoryStream(Encoding.UTF8.GetBytes(text3)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream3, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_028A:
					string fileNameWithoutExtension = Path.GetFileNameWithoutExtension(text);
					string text4 = string.Empty;
					if (!string.IsNullOrEmpty(tzpManager.CiteTAP) && tzpManager.ProgramName != fileNameWithoutExtension)
					{
						text4 = tzpManager.CiteTAP;
					}
					else
					{
						text4 = tzpManager.TAP;
					}
					if (string.IsNullOrEmpty(text4))
					{
						goto IL_0935;
					}
					text4 = text4.Replace("><![CDATA[", ">\r\n<![CDATA[");
					text4 = text4.Replace("]]><", "]]>\r\n<");
					using (MemoryStream memoryStream4 = new MemoryStream(Encoding.UTF8.GetBytes(text4)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream4, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_0351:
					string text5 = tzpManager.UpdatedTAP;
					if (string.IsNullOrEmpty(text5))
					{
						goto IL_0935;
					}
					text5 = text5.Replace("><![CDATA[", ">\r\n<![CDATA[");
					text5 = text5.Replace("]]><", "]]>\r\n<");
					using (MemoryStream memoryStream5 = new MemoryStream(Encoding.UTF8.GetBytes(text5)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream5, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_03E3:
					if (string.IsNullOrEmpty(tzpManager.DIFF_SRC))
					{
						goto IL_0935;
					}
					string extension2 = Path.GetExtension(Path.GetFileNameWithoutExtension(text));
					string text6;
					if ((text6 = extension2) == null)
					{
						goto IL_0935;
					}
					if (!(text6 == ".4gl"))
					{
						if (!(text6 == ".tap"))
						{
							goto IL_0935;
						}
					}
					else
					{
						using (MemoryStream memoryStream6 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.DIFF_SRC)))
						{
							zipOutputStream.PutNextEntry(new ZipEntry(text)
							{
								DateTime = DateTime.Now
							});
							StreamUtils.Copy(memoryStream6, zipOutputStream, new byte[4096]);
							zipOutputStream.CloseEntry();
							goto IL_0935;
						}
					}
					if (string.IsNullOrEmpty(tzpManager.DIFF_TAP.ToString()))
					{
						goto IL_0935;
					}
					text4 = tzpManager.DIFF_TAP.ToString();
					text4 = text4.Replace("><![CDATA[", ">\r\n<![CDATA[");
					text4 = text4.Replace("]]><", "]]>\r\n<");
					using (MemoryStream memoryStream7 = new MemoryStream(Encoding.UTF8.GetBytes(text4)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream7, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_0527:
					if (string.IsNullOrEmpty(tzpManager.TGL))
					{
						goto IL_0935;
					}
					using (MemoryStream memoryStream8 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.TGL)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream8, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_0593:
					using (MemoryStream memoryStream9 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.FullCode)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream9, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_05EF:
					if (string.IsNullOrEmpty(tzpManager.FormLocalizedStrings))
					{
						goto IL_0935;
					}
					using (MemoryStream memoryStream10 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.FormLocalizedStrings)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream10, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_065B:
					if (string.IsNullOrEmpty(tzpManager.GeneroFormString))
					{
						goto IL_0935;
					}
					using (MemoryStream memoryStream11 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.GeneroFormString)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream11, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_06C7:
					string fileNameWithoutExtension2 = Path.GetFileNameWithoutExtension(text);
					string text8;
					if (fileNameWithoutExtension2.Equals(tzpManager.SpecificationInfo.Key.Program))
					{
						string text7 = tzpManager.Tsd;
						text7 = text7.Replace("><![CDATA[", ">\r\n<![CDATA[");
						text7 = text7.Replace("]]><", "]]>\r\n<");
						text8 = text7;
					}
					else
					{
						text8 = tzpManager.SpecificationInfo.CiteSTD.ToString();
					}
					using (MemoryStream memoryStream12 = new MemoryStream(Encoding.UTF8.GetBytes(text8)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream12, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_078D:
					text8 = tzpManager.UpdatedTSD;
					text8 = text8.Replace("><![CDATA[", ">\r\n<![CDATA[");
					text8 = text8.Replace("]]><", "]]>\r\n<");
					using (MemoryStream memoryStream13 = new MemoryStream(Encoding.UTF8.GetBytes(text8)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream13, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_0813:
					using (MemoryStream memoryStream14 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.CustomActionDefaults)))
					{
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						StreamUtils.Copy(memoryStream14, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
						goto IL_0935;
					}
					IL_086F:
					flag = true;
					if (!string.IsNullOrEmpty(tzpManager.ElementBindings))
					{
						using (MemoryStream memoryStream15 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.ElementBindings)))
						{
							zipOutputStream.PutNextEntry(new ZipEntry(text)
							{
								DateTime = DateTime.Now
							});
							StreamUtils.Copy(memoryStream15, zipOutputStream, new byte[4096]);
							zipOutputStream.CloseEntry();
							goto IL_0935;
						}
						goto IL_08DB;
					}
					IL_0935:
					zipInputStream.CloseEntry();
					continue;
					IL_08DB:
					using (MemoryStream memoryStream16 = new MemoryStream())
					{
						zipInputStream.CopyTo(memoryStream16);
						zipOutputStream.PutNextEntry(new ZipEntry(text)
						{
							DateTime = DateTime.Now
						});
						memoryStream16.Position = 0L;
						StreamUtils.Copy(memoryStream16, zipOutputStream, new byte[4096]);
						zipOutputStream.CloseEntry();
					}
					goto IL_0935;
				}
			}
			if (!flag && !string.IsNullOrEmpty(tzpManager.ElementBindings))
			{
				using (MemoryStream memoryStream17 = new MemoryStream(Encoding.UTF8.GetBytes(tzpManager.ElementBindings)))
				{
					zipOutputStream.PutNextEntry(new ZipEntry(tzpManager.ProgramName + ".bdx")
					{
						DateTime = DateTime.Now
					});
					StreamUtils.Copy(memoryStream17, zipOutputStream, new byte[4096]);
					zipOutputStream.CloseEntry();
				}
			}
			zipInputStream.IsStreamOwner = true;
			zipInputStream.Close();
			zipOutputStream.Close();
			memoryStream.Position = 0L;
			try
			{
				File.Delete(zipfile);
				FileStream fileStream = File.Create(zipfile);
				memoryStream.WriteTo(fileStream);
				fileStream.Close();
			}
			catch (Exception ex)
			{
				string text9 = string.Format((string)Application.Current.FindResource("Message_WriteTzpFail"), zipfile);
				DesignerMessageBox.Show(text9 + "\r\n" + ex.Message);
				return false;
			}
			memoryStream.Close();
			FileSecurity accessControl = File.GetAccessControl(zipfile);
			FileSystemAccessRule fileSystemAccessRule = new FileSystemAccessRule(new NTAccount("", "Everyone"), FileSystemRights.FullControl, AccessControlType.Allow);
			accessControl.AddAccessRule(fileSystemAccessRule);
			File.SetAccessControl(zipfile, accessControl);
			return true;
		}

		// Token: 0x0600076F RID: 1903 RVA: 0x000221A8 File Offset: 0x000203A8
		public static string SeekReleaseVersion(string tzpFile)
		{
			using (FileStream fileStream = new FileStream(tzpFile, FileMode.Open, FileAccess.Read))
			{
				using (ZipFile zipFile = new ZipFile(fileStream))
				{
					ZipEntry entry = zipFile.GetEntry("ver");
					if (entry != null)
					{
						StreamReader streamReader = new StreamReader(zipFile.GetInputStream(entry));
						return streamReader.ReadLine();
					}
				}
			}
			throw new Exception(Application.Current.FindResource("Message_VersionNotFoundInTZS") as string);
		}
	}
}
