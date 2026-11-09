using System;
using System.Deployment.Application;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Text;

namespace SpecDesignerCommon.Logger
{
	// Token: 0x02000117 RID: 279
	public class DSCLogger
	{
		// Token: 0x1700029F RID: 671
		// (get) Token: 0x060009EC RID: 2540 RVA: 0x0003172B File Offset: 0x0002F92B
		private static string LOGPATH
		{
			get
			{
				if (SettingManager.Get().CurrentSetting != null)
				{
					return string.Format("{0}\\log\\", SettingManager.Get().CurrentSetting.Connection.Workspace);
				}
				return "C:\\TT\\log\\";
			}
		}

		// Token: 0x060009ED RID: 2541 RVA: 0x00031760 File Offset: 0x0002F960
		public static string GetUserAppDataPath()
		{
			return Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "DSC\\SpecDesigner");
		}

		// Token: 0x060009EE RID: 2542 RVA: 0x00031780 File Offset: 0x0002F980
		private static string GetVersionInfo()
		{
			if (ApplicationDeployment.IsNetworkDeployed)
			{
				ApplicationDeployment currentDeployment = ApplicationDeployment.CurrentDeployment;
				return currentDeployment.CurrentVersion.ToString();
			}
			return Assembly.GetEntryAssembly().GetName().Version.ToString();
		}

		// Token: 0x060009EF RID: 2543 RVA: 0x000317BC File Offset: 0x0002F9BC
		private static void GetSourceException(StreamWriter sw, Exception ce)
		{
			sw.WriteLine(string.Format("Message：{0}", ce.Message));
			sw.WriteLine(string.Format("Source：{0}", ce.Source));
			sw.WriteLine(string.Format("TargetSite ：{0}", ce.TargetSite));
			string[] array = ((ce.StackTrace != null) ? ce.StackTrace.Split(new string[] { "\r\n" }, StringSplitOptions.RemoveEmptyEntries) : new string[] { string.Empty });
			string text = ((array.Count<string>() > 0) ? array[0] : string.Empty);
			sw.WriteLine(string.Format("StackTrace ：{0}", text));
			if (ce.InnerException != null)
			{
				sw.WriteLine("-----------");
				DSCLogger.GetSourceException(sw, ce.InnerException);
			}
		}

		// Token: 0x060009F0 RID: 2544 RVA: 0x00031888 File Offset: 0x0002FA88
		public static void Write(string message, string programName)
		{
			if (!Directory.Exists(DSCLogger.LOGPATH))
			{
				Directory.CreateDirectory(DSCLogger.LOGPATH);
			}
			string text = string.Format("{0}errlog_{1}.log", DSCLogger.LOGPATH, DateTime.Now.ToString("yyyyMMdd"));
			StreamWriter streamWriter = new StreamWriter(text, true, Encoding.UTF8);
			streamWriter.WriteLine(string.Format("File: {0}", programName));
			streamWriter.WriteLine(string.Format("Version：{0}", DSCLogger.GetVersionInfo()));
			streamWriter.WriteLine(string.Format("Date: {0}", DateTime.UtcNow.ToLocalTime()));
			streamWriter.WriteLine(string.Format("Message: {0}", message));
			streamWriter.WriteLine("====================================");
			streamWriter.Close();
		}

		// Token: 0x060009F1 RID: 2545 RVA: 0x00031945 File Offset: 0x0002FB45
		public static void Write(Exception ce)
		{
			DSCLogger.Write(ce, string.Empty, TzpType.None);
		}

		// Token: 0x060009F2 RID: 2546 RVA: 0x00031954 File Offset: 0x0002FB54
		public static void Write(Exception ce, string programName, TzpType type)
		{
			if (!Directory.Exists(DSCLogger.LOGPATH))
			{
				Directory.CreateDirectory(DSCLogger.LOGPATH);
			}
			string text = string.Format("{0}errlog_{1}.log", DSCLogger.LOGPATH, DateTime.Now.ToString("yyyyMMdd"));
			StreamWriter streamWriter = new StreamWriter(text, true, Encoding.UTF8);
			streamWriter.WriteLine(">> ERROR LOG：{0} <<", DateTime.UtcNow.ToLocalTime());
			streamWriter.WriteLine(string.Format("File：{0}", programName));
			streamWriter.WriteLine(string.Format("File type：{0}", type));
			streamWriter.WriteLine(string.Format("Version：{0}", DSCLogger.GetVersionInfo()));
			DSCLogger.GetSourceException(streamWriter, ce);
			streamWriter.WriteLine("====================================");
			streamWriter.Close();
		}
	}
}
