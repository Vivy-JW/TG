using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net.Sockets;
using System.Windows;
using System.Windows.Input;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Logger;
using SpecDesignerCommon.Site.ViewModels;
using SpecDesignerCommon.ViewModel;
using SpecDesignerPreference;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x020000A1 RID: 161
	public class ConnectionManager
	{
		// Token: 0x06000684 RID: 1668 RVA: 0x0001CB14 File Offset: 0x0001AD14
		private static bool CheckGDC()
		{
			for (;;)
			{
				if (Process.GetProcesses().Any<Process>((Process p) => p.ProcessName.Equals("gdc", StringComparison.InvariantCultureIgnoreCase)))
				{
					break;
				}
				bool flag;
				using (CheckGDCWindow checkGDCWindow = new CheckGDCWindow())
				{
					checkGDCWindow.ShowDialog();
					switch (checkGDCWindow.CheckOption)
					{
					case CheckOptionEnum.Cancel:
						flag = false;
						break;
					case CheckOptionEnum.Continue:
						flag = true;
						break;
					default:
						continue;
					}
				}
				return flag;
			}
			return true;
		}

		// Token: 0x06000685 RID: 1669 RVA: 0x0001CB98 File Offset: 0x0001AD98
		private static bool OpenTelnetChannel()
		{
			if (!ConnectionInfo.This.IsLogin && new LoginWindow().ShowDialog() == false)
			{
				return false;
			}
			SettingModel currentSetting = SettingManager.Get().CurrentSetting;
			return ConnectionManager.OpenTelnetChannel(currentSetting.Connection.IP, int.Parse(currentSetting.Connection.Port), currentSetting.Connection.Login, currentSetting.Connection.Password, currentSetting.Connection.Protocol, currentSetting.Connection.ExtraCommand, currentSetting.Connection.LoginPrompt, currentSetting.Connection.PasswordPrompt);
		}

		// Token: 0x06000686 RID: 1670 RVA: 0x0001CC44 File Offset: 0x0001AE44
		private static bool OpenTelnetChannel(string IP, int Port, string UserName, string Password, string protocol, string extraCommand, string loginPrompt, string passwordPrompt)
		{
			string text = string.Format("Connection Information: {0}://{1}:{2}", protocol, IP, Port);
			bool flag = false;
			if (ConnectionInfo.This.Telnet == null)
			{
				if (protocol != null)
				{
					if (!(protocol == "telnet"))
					{
						if (protocol == "ssh")
						{
							ConnectionInfo.This.Telnet = new SshNetwork(IP, Port, 15);
						}
					}
					else
					{
						ConnectionInfo.This.Telnet = new Telnet(IP, Port, 15);
					}
				}
				if (ConnectionInfo.This.Telnet != null)
				{
					ConnectionInfo.This.Telnet.LoginPrompt = loginPrompt;
					ConnectionInfo.This.Telnet.PasswordPrompt = passwordPrompt;
				}
			}
			if (ConnectionInfo.This.SLW == null)
			{
				ConnectionInfo.This.SLW = new ServerLog();
			}
			ConnectionInfo.This.SLW.StartListening();
			ConnectionInfo.This.LoginUser = UserName;
			ConnectionInfo.This.ServerIP = IP;
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				if (!ConnectionInfo.This.Telnet.IsConnected)
				{
					ConnectionInfo.This.IsLogin = ConnectionInfo.This.Telnet.Connect(UserName, Password);
				}
				else
				{
					ConnectionInfo.This.IsLogin = true;
				}
				flag = ConnectionInfo.This.IsLogin;
			}
			catch (SocketException ex)
			{
				ConnectionInfo.This.IsLogin = false;
				ConnectionManager.CloseTelnetChannel();
				flag = false;
				string text2 = string.Format("Error Message：{0}\nError Code：{1}", ex.Message, ex.SocketErrorCode);
				DSCLogger.Write(string.Format("{0}\n{1}", text2, text), string.Empty);
				DesignerMessageBox.Show(string.Format("{0}{1}{1}{2}", Application.Current.FindResource("Message_SocketException") as string, Environment.NewLine, ex.Message), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
			catch (TimeoutException ex2)
			{
				ConnectionInfo.This.IsLogin = false;
				ConnectionManager.CloseTelnetChannel();
				flag = false;
				DSCLogger.Write(string.Format("{0}\n{1}", ex2.Message, text), string.Empty);
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SocketTimeoutException") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
			catch (Exception ex3)
			{
				ConnectionInfo.This.IsLogin = false;
				DSCLogger.Write(string.Format("{0}\n{1}", ex3.Message, text), string.Empty);
				DesignerMessageBox.Show(string.Format("{0}{1}{1}{2}", Application.Current.FindResource("Message_SocketException") as string, Environment.NewLine, ex3.Message), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			if (flag && !string.IsNullOrEmpty(extraCommand))
			{
				ConnectionInfo.This.Telnet.Send(extraCommand);
			}
			return flag;
		}

		// Token: 0x06000687 RID: 1671 RVA: 0x0001CF8C File Offset: 0x0001B18C
		public static void CloseTelnetChannel()
		{
			if (ConnectionInfo.This.SLW != null)
			{
				ConnectionInfo.This.SLW.StopListening();
			}
			if (ConnectionInfo.This.Telnet == null)
			{
				return;
			}
			ConnectionInfo.This.Telnet.Close();
			ConnectionInfo.This.Telnet = null;
			ConnectionInfo.This.IsLogin = false;
		}

		// Token: 0x06000688 RID: 1672 RVA: 0x0001CFE8 File Offset: 0x0001B1E8
		public static bool VerifyAccount(string UserName, string Password)
		{
			bool flag = false;
			ITTNetwork ittnetwork = null;
			ConnectionSetting connection = SettingManager.Get().CurrentSetting.Connection;
			string protocol;
			if ((protocol = connection.Protocol) != null)
			{
				if (!(protocol == "telnet"))
				{
					if (protocol == "ssh")
					{
						ittnetwork = new SshNetwork(connection.IP, int.Parse(connection.Port), 15);
					}
				}
				else
				{
					ittnetwork = new Telnet(connection.IP, int.Parse(connection.Port), 15);
				}
			}
			ittnetwork.LoginPrompt = connection.LoginPrompt;
			ittnetwork.PasswordPrompt = connection.PasswordPrompt;
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				flag = ittnetwork.Connect(UserName, Password);
			}
			catch (SocketException ex)
			{
				ittnetwork.Close();
				flag = false;
				string text = string.Format("Error Message：{0}\nError Code：{1}", ex.Message, ex.SocketErrorCode);
				DSCLogger.Write(text, string.Empty);
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SocketException") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
			catch (TimeoutException ex2)
			{
				ittnetwork.Close();
				flag = false;
				DSCLogger.Write(ex2.Message, string.Empty);
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SocketTimeoutException") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
			catch (Exception ex3)
			{
				ittnetwork.Close();
				flag = false;
				DSCLogger.Write(ex3.Message, string.Empty);
				DesignerMessageBox.Show(ex3.Message, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			if (flag)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Connect_topstdLoggedIn") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
			}
			return flag;
		}

		// Token: 0x06000689 RID: 1673 RVA: 0x0001D244 File Offset: 0x0001B444
		private static bool Downloads(string targetDIR, TzpType type)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				PreferenceModel preferenceModel = new PreferenceModel(PreferenceManager.Current.Settings.ToString());
				string text = string.Format("r.r adzp050 DIR '{0}\\' {1} {2} '{3}'", new object[]
				{
					targetDIR,
					type.Description(),
					preferenceModel.ServiceCloudLogin,
					preferenceModel.ServiceCloudPassword
				});
				ConnectionInfo.This.Telnet.Send(text);
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x0600068A RID: 1674 RVA: 0x0001D314 File Offset: 0x0001B514
		private static bool Downloads(string targetDIR, TzpType type, bool SimpleDownloadDialog)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				PreferenceModel preferenceModel = new PreferenceModel(PreferenceManager.Current.Settings.ToString());
				string text = preferenceModel.ServiceCloudLogin;
				if (string.IsNullOrEmpty(text))
				{
					text = "''";
				}
				string text2 = preferenceModel.ServiceCloudPassword;
				if (string.IsNullOrEmpty(text2))
				{
					text2 = "";
				}
				string text3 = "N";
				if (SimpleDownloadDialog)
				{
					text3 = "Y";
				}
				string text4 = string.Format("r.r adzp050 DIR '{0}\\' {1} {2} '{3}' '{4}'", new object[]
				{
					targetDIR,
					type.Description(),
					text,
					text2,
					text3
				});
				ConnectionInfo.This.Telnet.Send(text4);
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x0600068B RID: 1675 RVA: 0x0001D420 File Offset: 0x0001B620
		public static bool UploadReportTemplate()
		{
			string text = string.Format("{0}\\", SettingManager.Get().CurrentSetting.Connection.Workspace.Replace("\\", "\\\\"));
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp080 '' '' '{0}' '{1}'", TzpType.Report.Description(), text));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x0600068C RID: 1676 RVA: 0x0001D4E4 File Offset: 0x0001B6E4
		public static bool UploadCode(string FullPath, string FileName, string ModuleName, string UploadType)
		{
			return ConnectionManager.UploadCode(FullPath, FileName, ModuleName, UploadType, string.Empty);
		}

		// Token: 0x0600068D RID: 1677 RVA: 0x0001D4F4 File Offset: 0x0001B6F4
		public static bool UploadCode_Modify(string FullPath, string FileName, string ModuleName, string UploadType)
		{
			return ConnectionManager.UploadCode(FullPath, FileName, ModuleName, UploadType, "Y");
		}

		// Token: 0x0600068E RID: 1678 RVA: 0x0001D504 File Offset: 0x0001B704
		public static bool UploadCode(string FullPath, string FileName, string ModuleName, string UploadType, string parameter)
		{
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp080 '{0}' {1} {2} '{3}' '{4}'", new object[] { FileName, ModuleName, UploadType, text, parameter }));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x0600068F RID: 1679 RVA: 0x0001D610 File Offset: 0x0001B810
		public static bool UploadSpec(string FullPath, string FileName, string ModuleName, string UploadType)
		{
			return ConnectionManager.UploadSpec(FullPath, FileName, ModuleName, UploadType, string.Empty);
		}

		// Token: 0x06000690 RID: 1680 RVA: 0x0001D620 File Offset: 0x0001B820
		public static bool UploadSpec_Modify(string FullPath, string FileName, string ModuleName, string UploadType)
		{
			return ConnectionManager.UploadSpec(FullPath, FileName, ModuleName, UploadType, "Y");
		}

		// Token: 0x06000691 RID: 1681 RVA: 0x0001D630 File Offset: 0x0001B830
		public static bool UploadSpec(string FullPath, string FileName, string ModuleName, string UploadType, string parameter)
		{
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp080 '{0}' {1} {2} '{3}' {4}", new object[] { FileName, ModuleName, UploadType, text, parameter }));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x06000692 RID: 1682 RVA: 0x0001D73C File Offset: 0x0001B93C
		public static bool SimpleFormUpload(string FullPath, string FileName, string ModuleName, string parameter)
		{
			bool flag = false;
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp051 'upload' '{0}' '{1}' '{2}'", text, FileName, ModuleName));
				flag = true;
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x06000693 RID: 1683 RVA: 0x0001D830 File Offset: 0x0001BA30
		public static bool SimpleFormDownload(string targetDIR)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				string text = string.Format("r.r adzp051 'download' '{0}\\'", targetDIR);
				ConnectionInfo.This.Telnet.SendUTF8(text);
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x06000694 RID: 1684 RVA: 0x0001D8C0 File Offset: 0x0001BAC0
		public static bool UploadIndFunctionSetup(string FileName, string FullPath)
		{
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp052 '{0}' '{1}'", FileName, text));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x06000695 RID: 1685 RVA: 0x0001D9B0 File Offset: 0x0001BBB0
		public static bool SpecificationDownload(string targetDIR)
		{
			return ConnectionManager.Downloads(targetDIR, TzpType.Form);
		}

		// Token: 0x06000696 RID: 1686 RVA: 0x0001D9B9 File Offset: 0x0001BBB9
		public static bool SpecificationDownload(string targetDIR, bool SimpleDownloadDialog)
		{
			return ConnectionManager.Downloads(targetDIR, TzpType.Form, SimpleDownloadDialog);
		}

		// Token: 0x06000697 RID: 1687 RVA: 0x0001D9C3 File Offset: 0x0001BBC3
		public static bool ProgramDownload(string targetDIR)
		{
			return ConnectionManager.Downloads(targetDIR, TzpType.Code);
		}

		// Token: 0x06000698 RID: 1688 RVA: 0x0001D9CC File Offset: 0x0001BBCC
		public static bool ReportTemplateDownload(string targetDIR)
		{
			return ConnectionManager.Downloads(targetDIR, TzpType.Report);
		}

		// Token: 0x06000699 RID: 1689 RVA: 0x0001D9D8 File Offset: 0x0001BBD8
		public static bool ProgramTest(string programName)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				ConnectionInfo.This.Telnet.Send(string.Format("r.r {0}", programName));
				ConnectionInfo.This.Telnet.Send("\r\n");
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x0600069A RID: 1690 RVA: 0x0001DA78 File Offset: 0x0001BC78
		public static bool ProgramDebug(string programName)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				string text = programName.Substring(0, 3);
				ConnectionInfo.This.Telnet.Send(string.Format("cd {0}/4gl; r.d {1}", text, programName));
			}
			catch (Exception)
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x0600069B RID: 1691 RVA: 0x0001DB10 File Offset: 0x0001BD10
		public static bool BasicDataUpdate()
		{
			SettingManager.Get().PrepareToLoadCommomData(false);
			string text = string.Format("adzp050 {0} DIR '{1}\\'", "TT", SettingManager.Get().CurrentSetting.Connection.Workspace);
			return ConnectionManager.RunProgram(text);
		}

		// Token: 0x0600069C RID: 1692 RVA: 0x0001DB54 File Offset: 0x0001BD54
		public static bool SelectBasicDataUpdate()
		{
			SettingManager.Get().PrepareToLoadCommomData(true);
			string text = string.Format("adzp020 '{0}\\'", SettingManager.Get().CurrentSetting.Connection.Workspace);
			return ConnectionManager.RunProgram(text);
		}

		// Token: 0x0600069D RID: 1693 RVA: 0x0001DB94 File Offset: 0x0001BD94
		public static bool SelectBasicDataUpdate(string parameter, string title)
		{
			SettingManager.Get().PrepareToLoadCommomData(true);
			string text = string.Format("adzp020 '{0}\\' {1} {2}", SettingManager.Get().CurrentSetting.Connection.Workspace, parameter, title);
			return ConnectionManager.RunProgram(text);
		}

		// Token: 0x0600069E RID: 1694 RVA: 0x0001DBD4 File Offset: 0x0001BDD4
		public static void ReGenerateBasicData()
		{
			SettingManager.Get().PrepareToLoadCommomData(false);
			string text = string.Format("adzp070 DIR '{0}\\'", SettingManager.Get().CurrentSetting.Connection.Workspace);
			ConnectionManager.RunProgram(text);
		}

		// Token: 0x0600069F RID: 1695 RVA: 0x0001DC14 File Offset: 0x0001BE14
		public static bool RunProgram(string programID)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				string text = string.Format("r.r {0}", programID);
				ConnectionInfo.This.Telnet.Send(text);
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x060006A0 RID: 1696 RVA: 0x0001DCA4 File Offset: 0x0001BEA4
		public static void ReBuildProgram(string progName, string ver, string type, string zone, string erpver)
		{
			ConnectionManager.RunProgram(string.Format("adzp062 {0} {1} {2} {3} {4}", new object[] { progName, ver, type, zone, erpver }));
		}

		// Token: 0x060006A1 RID: 1697 RVA: 0x0001DCE0 File Offset: 0x0001BEE0
		public static bool PreviewForm(string programName)
		{
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			try
			{
				Application.Current.MainWindow.Cursor = Cursors.Wait;
				ConnectionInfo.This.Telnet.Send(string.Format("r.p {0}", programName));
				ConnectionInfo.This.Telnet.Send("\r\n");
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				return false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return true;
		}

		// Token: 0x060006A2 RID: 1698 RVA: 0x0001DD80 File Offset: 0x0001BF80
		public static bool ProgramThisVersionModify(string FullPath, string FileName, string ModuleName, string UploadType)
		{
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp080 '{0}' {1} {2} '{3}' 'Y'", new object[] { FileName, ModuleName, UploadType, text }));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x060006A3 RID: 1699 RVA: 0x0001DE88 File Offset: 0x0001C088
		public static bool SpecThisVersionModify(string FullPath, string FileName, string ModuleName, string UploadType, string parameter)
		{
			string text = string.Format("{0}\\", Path.GetDirectoryName(FullPath)).Replace("\\", "\\\\");
			if (!File.Exists(FullPath))
			{
				string text2 = Application.Current.FindResource("Message_FileNotExist") as string;
				DesignerMessageBox.Show(string.Format(text2, FullPath), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				return false;
			}
			bool flag = true;
			if (!ConnectionManager.OpenTelnetChannel())
			{
				return false;
			}
			if (!ConnectionManager.CheckGDC())
			{
				return false;
			}
			Application.Current.MainWindow.Cursor = Cursors.Wait;
			try
			{
				ConnectionInfo.This.Telnet.SendUTF8(string.Format("r.r adzp080 '{0}' {1} {2} '{3}' {4} 'Y'", new object[] { FileName, ModuleName, UploadType, text, parameter }));
			}
			catch
			{
				ConnectionManager.CloseTelnetChannel();
				flag = false;
			}
			finally
			{
				Application.Current.MainWindow.Cursor = null;
			}
			return flag;
		}

		// Token: 0x060006A4 RID: 1700 RVA: 0x0001DF94 File Offset: 0x0001C194
		public static void SpecCheckIn(PackageKey ProgramKey)
		{
			string programName = SettingManager.Get().GetTzpManger(ProgramKey).ProgramName;
			string progType = SettingManager.Get().GetTzpManger(ProgramKey).ProgType;
			string moduleName = SettingManager.Get().GetTzpManger(ProgramKey).ModuleName;
			ConnectionManager.RunProgram(string.Format("adzp201 {0} SPEC {1} {2}", programName, progType, moduleName));
		}

		// Token: 0x060006A5 RID: 1701 RVA: 0x0001DFE8 File Offset: 0x0001C1E8
		public static void GeneralFunction(PackageKey ProgramKey, XmlElement Component)
		{
			TzpManager tzpManger = SettingManager.Get().GetTzpManger(ProgramKey);
			string programName = SettingManager.Get().GetTzpManger(ProgramKey).ProgramName;
			string ver = tzpManger.SpecificationInfo.Ver;
			string text = (tzpManger.Booking ? "Y" : "N");
			string name = Component.Name;
			string name2 = tzpManger.SpecificationInfo.FindContainer(Component).Name;
			string text2 = string.Format("adzi261 -PRGNO {0} -SPCVER {1} -DSTCNO {2} -DSTCON {3} -SPCCKO {4}", new object[] { programName, ver, name, name2, text });
			Trace.WriteLine(text2);
			ConnectionManager.RunProgram(text2);
		}

		// Token: 0x060006A6 RID: 1702 RVA: 0x0001E090 File Offset: 0x0001C290
		public static void GeneralValueFunc(PackageKey ProgramKey, XmlElement Component)
		{
			TzpManager tzpManger = SettingManager.Get().GetTzpManger(ProgramKey);
			string programName = SettingManager.Get().GetTzpManger(ProgramKey).ProgramName;
			string ver = tzpManger.SpecificationInfo.Ver;
			string text = (tzpManger.Booking ? "Y" : "N");
			string name = Component.Name;
			string name2 = tzpManger.SpecificationInfo.FindContainer(Component).Name;
			string text2 = string.Format("adzi262 -PRGNO {0} -SPCVER {1} -DSTCNO {2} -DSTCON {3} -SPCCKO {4}", new object[] { programName, ver, name, name2, text });
			Trace.WriteLine(text2);
			ConnectionManager.RunProgram(text2);
		}
	}
}
