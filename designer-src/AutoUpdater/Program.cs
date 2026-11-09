using System;
using System.Diagnostics;
using System.IO;
using System.Windows;
using SpecDesigner.Controls.Controls;

namespace AutoUpdater
{
	// Token: 0x02000002 RID: 2
	internal class Program
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		private static void Main(string[] args)
		{
			if (args.Length != 3)
			{
				Environment.Exit(0);
			}
			string text = string.Empty;
			string text2 = string.Empty;
			string text3 = string.Empty;
			text = args[0];
			text2 = args[1];
			text3 = args[2];
			Program.KillDesigner(text2);
			string directoryName = Path.GetDirectoryName(text);
			string directoryName2 = Path.GetDirectoryName(Environment.GetCommandLineArgs()[0]);
			Program.CopyUpgradeFiles(directoryName2, directoryName);
			Program.WakeApplication(text, text3);
			Environment.Exit(0);
		}

		// Token: 0x06000002 RID: 2 RVA: 0x000020B8 File Offset: 0x000002B8
		private static void WakeApplication(string exeProgram, string uid)
		{
			try
			{
				ProcessStartInfo processStartInfo = new ProcessStartInfo(exeProgram);
				new Process
				{
					StartInfo = processStartInfo
				}.Start();
			}
			catch (Exception ex)
			{
				DesignerMessageBox.Show((Application.Current.FindResource("Message_StartError") as string) + ex.Message, Application.Current.FindResource("Message_Error") as string);
			}
		}

		// Token: 0x06000003 RID: 3 RVA: 0x00002130 File Offset: 0x00000330
		private static void CopyUpgradeFiles(string source, string target)
		{
			try
			{
				string[] files = Directory.GetFiles(source);
				foreach (string text in files)
				{
					if (!(Path.GetFileName(text) == "AutoUpdater.exe"))
					{
						Path.Combine(source, text);
						string text2 = Path.Combine(target, Path.GetFileName(text));
						File.Copy(text, text2, true);
					}
				}
			}
			catch (Exception ex)
			{
				DesignerMessageBox.Show(ex.Message);
			}
		}

		// Token: 0x06000004 RID: 4 RVA: 0x000021BC File Offset: 0x000003BC
		private static void CopyFiles(string srcFolder, string dstFolder)
		{
			Program.proc = Process.Start(new ProcessStartInfo
			{
				UseShellExecute = false,
				CreateNoWindow = true,
				WorkingDirectory = srcFolder,
				RedirectStandardError = true,
				RedirectStandardOutput = true,
				RedirectStandardInput = true,
				Verb = "runas",
				FileName = "xcopy",
				Arguments = string.Format(" /R/Y/I \"{0}\" \"{1}\"", srcFolder, dstFolder)
			});
			Program.proc.ErrorDataReceived += Program.proc_ErrorDataReceived;
			Program.proc.BeginErrorReadLine();
			Program.proc.BeginOutputReadLine();
			Program.proc.WaitForExit();
		}

		// Token: 0x06000005 RID: 5 RVA: 0x00002260 File Offset: 0x00000460
		private static void proc_ErrorDataReceived(object sender, DataReceivedEventArgs e)
		{
			Program.proc.ErrorDataReceived -= Program.proc_ErrorDataReceived;
			if (e.Data == null)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_UpdateFinished") as string);
				return;
			}
			DesignerMessageBox.Show(string.Format("Error: {0}", e.Data));
		}

		// Token: 0x06000006 RID: 6 RVA: 0x000022BC File Offset: 0x000004BC
		private static void KillDesigner(string sourcePID)
		{
			Process[] processes = Process.GetProcesses();
			for (int i = 0; i < processes.Length; i++)
			{
				if (processes[i].ProcessName == "T100Designer")
				{
					Process process = processes[i];
					process.Kill();
					process.WaitForExit(2000);
					return;
				}
			}
		}

		// Token: 0x06000007 RID: 7 RVA: 0x00002308 File Offset: 0x00000508
		private static void KillOtherUpdate(string sourcePID)
		{
			Process currentProcess = Process.GetCurrentProcess();
			Process[] processes = Process.GetProcesses();
			for (int i = 0; i < processes.Length; i++)
			{
				if (processes[i].ProcessName == "AutoUpdater" && processes[i] != currentProcess)
				{
					Process process = processes[i];
					process.Kill();
					process.WaitForExit(2000);
					return;
				}
			}
		}

		// Token: 0x04000001 RID: 1
		private static Process proc;
	}
}
