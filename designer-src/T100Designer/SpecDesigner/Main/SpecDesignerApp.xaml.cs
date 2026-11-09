using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Threading;
using System.Windows;
using System.Windows.Markup;
using System.Windows.Threading;
using System.Xml;
using Microsoft.Practices.Prism.Events;
using Microsoft.Shell;
using SpecDesigner.Controls.Controls;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Exceptions;
using SpecDesignerCommon.Logger;
using SpecDesignerCommon.Site;
using SpecDesignerCommon.Site.ViewModels;
using SpecDesignerCommon.Views;
using SpecDesignerPreference;

namespace SpecDesigner.Main
{
	// Token: 0x0200002B RID: 43
	public partial class SpecDesignerApp : Application, ISingleInstanceApp
	{
		// Token: 0x17000097 RID: 151
		// (get) Token: 0x06000260 RID: 608 RVA: 0x0000B0B9 File Offset: 0x000092B9
		// (set) Token: 0x06000261 RID: 609 RVA: 0x0000B0C1 File Offset: 0x000092C1
		public string[] Args { get; private set; }

		// Token: 0x06000263 RID: 611 RVA: 0x0000B104 File Offset: 0x00009304
		private static void AppDomainExceptionHandler(object sender, UnhandledExceptionEventArgs args)
		{
			Exception ex = args.ExceptionObject as Exception;
			SpecDesignerApp.UnhandledExceptionProcessor(ex);
		}

		// Token: 0x06000264 RID: 612 RVA: 0x0000B1A0 File Offset: 0x000093A0
		protected override void OnStartup(StartupEventArgs e)
		{
			PresentationTraceSources.DataBindingSource.Switch.Level = SourceLevels.Critical;
			base.OnStartup(e);
			this.Args = e.Args;
			base.DispatcherUnhandledException += this.AppDispatcherUnhandledException;
			PreferenceManager.Current.LoadSettings();
			this.LoadLangs();
			SettingModel selectedSetting = null;
			if (this.Args.Count<string>() > 0)
			{
				switch (this.Args.Count<string>())
				{
				case 1:
					selectedSetting = SettingManager.Get().SiteManager.GetSettingByGuid(this.Args[0]);
					base.Properties["SiteManagerUID"] = ((selectedSetting != null) ? selectedSetting.UID : null);
					break;
				case 2:
					if (string.IsNullOrWhiteSpace(this.Args[0]))
					{
						selectedSetting = SettingManager.Get().SiteManager.ShowDialog(this.Args[1]);
					}
					else
					{
						selectedSetting = SettingManager.Get().SiteManager.GetSettingByGuid(this.Args[0]);
					}
					base.Properties["SiteManagerUID"] = ((selectedSetting != null) ? selectedSetting.UID : null);
					break;
				}
			}
			else
			{
				selectedSetting = SettingManager.Get().SiteManager.ShowDialog();
			}
			if (selectedSetting == null)
			{
				try
				{
					Application.Current.Shutdown();
				}
				catch
				{
				}
				return;
			}
			while (selectedSetting != null && !selectedSetting.Connection.IsValidate)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_ConnectionIsNotValidate") as string);
				selectedSetting = SettingManager.Get().SiteManager.ShowDialog();
			}
			if (selectedSetting == null)
			{
				Application.Current.Shutdown();
				return;
			}
			SettingManager.Get().CurrentSetting = selectedSetting;
			SpecDesignerSplashScreen specDesignerSplashScreen = new SpecDesignerSplashScreen(new SpecDesignerInit());
			bool? flag = null;
			try
			{
				flag = specDesignerSplashScreen.ShowDialog();
			}
			catch
			{
				flag = null;
			}
			if (flag == true)
			{
				SpecDesignerMainWindow specDesignerMainWindow = new SpecDesignerMainWindow();
				Application.Current.MainWindow = specDesignerMainWindow;
				Application.Current.MainWindow.Show();
				specDesignerMainWindow.Focus();
				if (!SettingManager.Get().CurrentSetting.IsCheckedRemoteVersion)
				{
					BackgroundWorker backgroundWorker = new BackgroundWorker();
					backgroundWorker.DoWork += delegate(object s, DoWorkEventArgs we)
					{
						this.Properties["SiteManagerUID"] = selectedSetting.UID;
						UpdateManager.This.IsForceUpdate = true;
						UpdateManager.This.checkFinished += this.CheckRemoteVersionFinished;
						UpdateManager.This.AppCastIP = SettingManager.Get().CurrentSetting.Connection.UpdateURL;
					};
					backgroundWorker.RunWorkerAsync();
				}
				if (this.Args.Count<string>() > 1)
				{
					SettingManager.Get().OpenSpecFiles(this.Args[1]);
					return;
				}
			}
			else
			{
				base.Shutdown();
			}
		}

		// Token: 0x06000265 RID: 613 RVA: 0x0000B488 File Offset: 0x00009688
		private void CheckRemoteVersionFinished(object sender, CheckVersionArgs e)
		{
			UpdateManager.This.checkFinished -= this.CheckRemoteVersionFinished;
			Version version = SettingManager.Get().Version;
			string text = string.Format("{0}.{1}", version.Major, version.Minor);
			UpdateManager updateManager = sender as UpdateManager;
			if (!e.HasError)
			{
				if (updateManager.Version != text)
				{
					string text2 = Application.Current.FindResource("Message_VersionInCompatibleWarning") as string;
					text2 = string.Format(text2, updateManager.Version, text);
					DesignerMessageBox.Show(text2, "Warning", MessageBoxButton.OK, MessageBoxImage.Exclamation);
				}
				return;
			}
			if (e.ExceptionMessage is FileNotFoundException)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_VersionNotFound") as string, Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
				return;
			}
			DesignerMessageBox.Show(string.Format("{0}{1}{2}", Application.Current.FindResource("Message_GetVersionButError") as string, Environment.NewLine, e.ExceptionMessage), Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
		}

		// Token: 0x06000266 RID: 614 RVA: 0x0000B5B0 File Offset: 0x000097B0
		private void LoadLangs()
		{
			string uilang = PreferenceManager.Current.Settings.UILang;
			CultureInfo cultureInfo = ((!string.IsNullOrEmpty(uilang)) ? CultureInfo.GetCultureInfo(uilang) : CultureInfo.CurrentCulture);
			Thread.CurrentThread.CurrentUICulture = cultureInfo;
			Thread.CurrentThread.CurrentCulture = cultureInfo;
			ResourceDictionary resourceDictionary = null;
			string text = "/SpecDesignerCommon;component/Langs/" + cultureInfo.Name + ".xaml";
			try
			{
				resourceDictionary = Application.LoadComponent(new Uri(text, UriKind.RelativeOrAbsolute)) as ResourceDictionary;
			}
			catch (Exception)
			{
				cultureInfo = CultureInfo.GetCultureInfo("en-US");
				Thread.CurrentThread.CurrentUICulture = cultureInfo;
				Thread.CurrentThread.CurrentCulture = cultureInfo;
			}
			if (resourceDictionary == null)
			{
				text = "/SpecDesignerCommon;component/Langs/en-US.xaml";
				resourceDictionary = Application.LoadComponent(new Uri(text, UriKind.RelativeOrAbsolute)) as ResourceDictionary;
			}
			Application.Current.Resources.MergedDictionaries.Add(resourceDictionary);
		}

		// Token: 0x06000267 RID: 615 RVA: 0x0000B68C File Offset: 0x0000988C
		private void AppDispatcherUnhandledException(object sender, DispatcherUnhandledExceptionEventArgs e)
		{
			this.ShowUnhandeledException(e);
		}

		// Token: 0x06000268 RID: 616 RVA: 0x0000B695 File Offset: 0x00009895
		private void ShowUnhandeledException(DispatcherUnhandledExceptionEventArgs e)
		{
			e.Handled = true;
			SpecDesignerApp.UnhandledExceptionProcessor(e.Exception);
		}

		// Token: 0x06000269 RID: 617 RVA: 0x0000B6AC File Offset: 0x000098AC
		public static void UnhandledExceptionProcessor(Exception ce)
		{
			string text = ce.Message + ((ce.InnerException != null) ? ("\n" + ce.InnerException.Message) : null);
			string text2 = string.Format("{0}\n\nError：\n{1}", Application.Current.FindResource("Message_ApplicationErrorMessage"), text);
			PackageKey currentProgram = EditorWorkspace.This.CurrentProgram;
			if (currentProgram != null)
			{
				DSCLogger.Write(ce, currentProgram.Program, currentProgram.PackType);
			}
			else
			{
				DSCLogger.Write(ce);
			}
			if (EditorWorkspace.This.CurrentOpening != null || currentProgram == null)
			{
				if (ce is ComplexException)
				{
					ComplexException ex = ce as ComplexException;
					DetailsMessageBox.Show(text, ex.DetailMessage, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				}
				else
				{
					DesignerMessageBox.Show(text, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				}
				EditorWorkspace.This.CloseOpening();
				return;
			}
			if (DesignerMessageBox.Show(text2, Application.Current.FindResource("Message_ApplicationErrorTitle") as string, MessageBoxButton.YesNoCancel, MessageBoxImage.Hand) == MessageBoxResult.No && DesignerMessageBox.Show(Application.Current.FindResource("Message_ExitBeforeSave") as string, Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.YesNoCancel, MessageBoxImage.Exclamation) == MessageBoxResult.Yes)
			{
				Application.Current.Shutdown();
			}
		}

		// Token: 0x0600026A RID: 618 RVA: 0x0000B804 File Offset: 0x00009A04
		public bool SignalExternalCommandLineArgs(IList<string> args)
		{
			if (args == null || args.Count == 0 || args[0].EndsWith("application"))
			{
				return false;
			}
			Uri uri = new Uri(args[0]);
			string localPath = uri.LocalPath;
			SettingManager.Get().OpenSpecFiles(localPath);
			return true;
		}

		// Token: 0x0600026B RID: 619 RVA: 0x0000B854 File Offset: 0x00009A54
		public static Version GetPublishedVersion()
		{
			XmlDocument xmlDocument = new XmlDocument();
			Assembly executingAssembly = Assembly.GetExecutingAssembly();
			string localPath = new Uri(executingAssembly.GetName().CodeBase).LocalPath;
			xmlDocument.Load(localPath + ".manifest");
			string text = string.Empty;
			if (xmlDocument.HasChildNodes)
			{
				text = xmlDocument.ChildNodes[1].ChildNodes[0].Attributes.GetNamedItem("version").Value.ToString();
			}
			return new Version(text);
		}

		// Token: 0x0600026C RID: 620 RVA: 0x0000B8D9 File Offset: 0x00009AD9
		protected override void OnExit(ExitEventArgs e)
		{
			ConnectionInfo.This.Close();
			base.OnExit(e);
		}

		// Token: 0x04000165 RID: 357
		private const string Unique = "SpecDesigner";

		// Token: 0x04000166 RID: 358
		private IEventAggregator DesignerEventAggregator = new EventAggregator();
	}
}
