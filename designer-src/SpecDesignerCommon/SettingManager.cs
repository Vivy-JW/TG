using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Threading;
using System.Windows;
using System.Xml.Linq;
using ICSharpCode.SharpZipLib.Zip;
using Microsoft.Win32;
using SpecDesigner.Controls.Controls;
using SpecDesigner.CustomException;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Exceptions;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Site;
using SpecDesignerCommon.Site.ViewModels;
using SpecDesignerCommon.ViewModel;
using SpecDesignerCommon.Views;
using UndoRedoFramework;

namespace SpecDesignerCommon
{
	// Token: 0x02000064 RID: 100
	public class SettingManager
	{
		// Token: 0x170000FD RID: 253
		// (get) Token: 0x060003B5 RID: 949 RVA: 0x00010DF1 File Offset: 0x0000EFF1
		// (set) Token: 0x060003B6 RID: 950 RVA: 0x00010DF9 File Offset: 0x0000EFF9
		public SettingModel CurrentSetting { get; set; }

		// Token: 0x170000FE RID: 254
		// (get) Token: 0x060003B7 RID: 951 RVA: 0x00010E02 File Offset: 0x0000F002
		public SiteManager SiteManager
		{
			get
			{
				if (this._siteManager == null)
				{
					this._siteManager = new SiteManager();
				}
				return this._siteManager;
			}
		}

		// Token: 0x060003B8 RID: 952 RVA: 0x00010E1D File Offset: 0x0000F01D
		public static SettingManager Get()
		{
			if (SettingManager.settingManager == null)
			{
				SettingManager.settingManager = new SettingManager();
			}
			return SettingManager.settingManager;
		}

		// Token: 0x060003B9 RID: 953 RVA: 0x00010E35 File Offset: 0x0000F035
		public TzpManager GetTzpManger(PackageKey key)
		{
			if (this.tzpMap.ContainsKey(key))
			{
				TzpManager.Current = this.tzpMap[key];
				return this.tzpMap[key];
			}
			return null;
		}

		// Token: 0x060003BA RID: 954 RVA: 0x00010E64 File Offset: 0x0000F064
		public UndoRedoManager GetUndoRedoManager(PackageKey key)
		{
			if (this.undoRedoManagerMap.ContainsKey(key))
			{
				return this.undoRedoManagerMap[key];
			}
			throw new Exception("No UndoRedoManager");
		}

		// Token: 0x060003BB RID: 955 RVA: 0x00010E8B File Offset: 0x0000F08B
		public bool CheckUndoRedoManager(PackageKey key)
		{
			return this.undoRedoManagerMap.ContainsKey(key);
		}

		// Token: 0x1400000E RID: 14
		// (add) Token: 0x060003BC RID: 956 RVA: 0x00010E9C File Offset: 0x0000F09C
		// (remove) Token: 0x060003BD RID: 957 RVA: 0x00010ED4 File Offset: 0x0000F0D4
		public event EventHandler CommonDataLoaded;

		// Token: 0x060003BE RID: 958 RVA: 0x00010F0C File Offset: 0x0000F10C
		private SettingManager()
		{
			this.specReferFiles = new string[]
			{
				"mta/core-br.spec", "mta/mod-fd.spec", "mta/tiptop.4ad", "4tb/toolbar_i.4tb", "mta/subroutines.xml", "mta/checks.xml", "mta/datatypes.xml", "mta/items.xml", "mta/libraries.xml", "mta/messages.xml",
				"mta/prog_rel.xml", "mta/tables.xml", "mta/4gl.xml", "mta/zooms.xml", "mta/top_global.inc", "mta/tsd.xsd", "mta/ver", "mta/code_sample.xml", "mta/subinfo.xml", "mta/top_global.xml",
				"mta/languages.xml"
			};
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.CloseFile));
		}

		// Token: 0x060003BF RID: 959 RVA: 0x0001102C File Offset: 0x0000F22C
		public void PrepareToLoadCommomData(bool bypass)
		{
			if (this._fileSystemWatcher == null)
			{
				this._fileSystemWatcher = new FileSystemWatcher(this.CurrentSetting.Connection.Workspace);
				this._fileSystemWatcher.Filter = this.TTzip;
				this._fileSystemWatcher.IncludeSubdirectories = false;
				this._fileSystemWatcher.NotifyFilter = NotifyFilters.FileName | NotifyFilters.DirectoryName | NotifyFilters.LastWrite | NotifyFilters.LastAccess | NotifyFilters.CreationTime;
				if (bypass)
				{
					this._fileSystemWatcher.Created -= this.OnPickModuleFileChanged;
					this._fileSystemWatcher.Changed -= this.OnPickModuleFileChanged;
					this._fileSystemWatcher.Created += this.OnPickModuleFileChanged;
					this._fileSystemWatcher.Changed += this.OnPickModuleFileChanged;
				}
				else
				{
					this._fileSystemWatcher.Created -= this.OnFileChanged;
					this._fileSystemWatcher.Changed -= this.OnFileChanged;
					this._fileSystemWatcher.Created += this.OnFileChanged;
					this._fileSystemWatcher.Changed += this.OnFileChanged;
				}
				this._fileSystemWatcher.EnableRaisingEvents = true;
			}
		}

		// Token: 0x060003C0 RID: 960 RVA: 0x00011154 File Offset: 0x0000F354
		private void OnFileChanged(object sender, FileSystemEventArgs e)
		{
			try
			{
				using (File.OpenRead(e.FullPath))
				{
					this._fileSystemWatcher.Created -= this.OnFileChanged;
					this._fileSystemWatcher.Changed -= this.OnFileChanged;
				}
			}
			catch
			{
				return;
			}
			this.UnzipThenLoad(false);
		}

		// Token: 0x060003C1 RID: 961 RVA: 0x000111D0 File Offset: 0x0000F3D0
		private void OnPickModuleFileChanged(object sender, FileSystemEventArgs e)
		{
			try
			{
				using (File.OpenRead(e.FullPath))
				{
					this._fileSystemWatcher.Created -= this.OnPickModuleFileChanged;
					this._fileSystemWatcher.Changed -= this.OnPickModuleFileChanged;
				}
			}
			catch
			{
				return;
			}
			this.UnzipThenLoad(true);
		}

		// Token: 0x060003C2 RID: 962 RVA: 0x000113B8 File Offset: 0x0000F5B8
		private void UnzipThenLoad(bool bypass)
		{
			string workspace = this.CurrentSetting.Connection.Workspace;
			string text = Path.Combine(workspace, this.TTzip);
			string text2 = string.Format("{0}.{1}", this.TTzip, DateTime.Now.Ticks);
			string text3 = Path.Combine(workspace, text2);
			try
			{
				File.Copy(text, text3);
			}
			catch (FileNotFoundException)
			{
				Application.Current.Dispatcher.Invoke(new Action(delegate
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_BasicDataNotFound") as string, new object[0]), Application.Current.FindResource("menu_BasicDataUpdate") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				}), new object[0]);
				return;
			}
			catch (Exception ex)
			{
				Exception ce2 = ex;
				Application.Current.Dispatcher.Invoke(new Action(delegate
				{
					string text5 = Application.Current.FindResource("Message_BasicDataUpdateFailed") as string;
					DesignerMessageBox.Show(string.Format(text5, ce2.Message), Application.Current.FindResource("menu_BasicDataUpdate") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				}), new object[0]);
				return;
			}
			try
			{
				if (!SettingManager.Unzip(text3, workspace))
				{
					throw new Exception(Application.Current.FindResource("Message_BasicDataUnzipFailed") as string);
				}
				if (bypass)
				{
					string text4 = Path.Combine(workspace, "mta", "bypass");
					if (File.Exists(text4))
					{
						File.Delete(text4);
						this._fileSystemWatcher.Dispose();
						this._fileSystemWatcher = null;
						return;
					}
				}
				this.LoadCommonData();
				Application.Current.Dispatcher.BeginInvoke(new Action(delegate
				{
					ProgressBar.Instance.Hide();
					using (AutoCloseDialog autoCloseDialog = new AutoCloseDialog())
					{
						autoCloseDialog.Message = Application.Current.FindResource("Message_AfterUpdateBasicData") as string;
						autoCloseDialog.Title = Application.Current.FindResource("menu_BasicDataUpdate") as string;
						autoCloseDialog.ImprovedShowDialog();
					}
				}), new object[0]);
				if (this.CommonDataLoaded != null)
				{
					this.CommonDataLoaded(this, EventArgs.Empty);
				}
			}
			catch (Exception ex2)
			{
				Exception ce = ex2;
				Application.Current.Dispatcher.BeginInvoke(new Action(delegate
				{
					ProgressBar.Instance.Hide();
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_BasicDataLoadFailed") as string, ce.Message), Application.Current.FindResource("menu_BasicDataUpdate") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
				}), new object[0]);
			}
			try
			{
				File.Delete(text);
			}
			catch (Exception ex3)
			{
				Trace.WriteLine(ex3.Message);
			}
			this._fileSystemWatcher.Dispose();
			this._fileSystemWatcher = null;
		}

		// Token: 0x060003C3 RID: 963 RVA: 0x000115D8 File Offset: 0x0000F7D8
		private static bool Unzip(string file, string dir)
		{
			try
			{
				new FastZip
				{
					CreateEmptyDirectories = true
				}.ExtractZip(file, dir, FastZip.Overwrite.Always, null, null, null, true);
			}
			catch (Exception)
			{
				return false;
			}
			File.Delete(file);
			return true;
		}

		// Token: 0x170000FF RID: 255
		// (get) Token: 0x060003C4 RID: 964 RVA: 0x00011620 File Offset: 0x0000F820
		// (set) Token: 0x060003C5 RID: 965 RVA: 0x00011628 File Offset: 0x0000F828
		public string LastOpenPath
		{
			get
			{
				return this._lastOpenPath;
			}
			set
			{
				this._lastOpenPath = value;
			}
		}

		// Token: 0x060003C6 RID: 966 RVA: 0x00011634 File Offset: 0x0000F834
		public void OpenFile()
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			if (string.IsNullOrEmpty(this.LastOpenPath))
			{
				openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			}
			else
			{
				openFileDialog.InitialDirectory = this.LastOpenPath;
			}
			openFileDialog.Filter = "Spec Files (*.tzc, *.tzs, *.tzd, *.tzr, *.tzg, *.tzt)|*.tzc;*.tzs;*.tzd;*.tzr;*.tzg;*.tzt;|All Files (*.*)|*.*";
			if (openFileDialog.ShowDialog() == true)
			{
				this.LastOpenPath = openFileDialog.FileName.Replace(openFileDialog.SafeFileName, "");
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text in fileNames)
				{
					this.OpenSpecFiles(text);
				}
			}
		}

		// Token: 0x060003C7 RID: 967 RVA: 0x000116F0 File Offset: 0x0000F8F0
		public void OpenDiffFile()
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			if (string.IsNullOrEmpty(this.LastOpenPath))
			{
				openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			}
			else
			{
				openFileDialog.InitialDirectory = this.LastOpenPath;
			}
			openFileDialog.Filter = "Spec Files (*.tzx)|*.tzx;|All Files (*.*)|*.*";
			if (openFileDialog.ShowDialog() == true)
			{
				this.LastOpenPath = openFileDialog.FileName.Replace(openFileDialog.SafeFileName, "");
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text in fileNames)
				{
					this.OpenSpecFiles(text);
				}
			}
		}

		// Token: 0x060003C8 RID: 968 RVA: 0x000117AC File Offset: 0x0000F9AC
		public void OpenFile(int type)
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			if (string.IsNullOrEmpty(this.LastOpenPath))
			{
				openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			}
			else
			{
				openFileDialog.InitialDirectory = this.LastOpenPath;
			}
			string text = "Spec Files (*.tzc, *.tzs, *.tzd, *.tzr, *.tzg, *.tzt)|*.tzc;*.tzs;*.tzd;*.tzr;*.tzg;*.tzt;|All Files (*.*)|*.*";
			switch (type)
			{
			case 1:
				text = string.Format("{0}{1}", "TZS Files (*.tzs)|*.tzs;|", text);
				break;
			case 2:
				text = string.Format("{0}{1}", "TZC Files (*.tzc)|*.tzc;|", text);
				break;
			}
			openFileDialog.Filter = text;
			if (openFileDialog.ShowDialog() == true)
			{
				this.LastOpenPath = openFileDialog.FileName.Replace(openFileDialog.SafeFileName, "");
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text2 in fileNames)
				{
					this.OpenSpecFiles(text2);
				}
			}
		}

		// Token: 0x060003C9 RID: 969 RVA: 0x000118A8 File Offset: 0x0000FAA8
		public void OpenSimpleFormFile()
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			if (string.IsNullOrEmpty(this.LastOpenPath))
			{
				openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			}
			else
			{
				openFileDialog.InitialDirectory = this.LastOpenPath;
			}
			openFileDialog.Filter = "Spec Files (*.tzv)|*.tzv;|All Files (*.*)|*.*";
			if (openFileDialog.ShowDialog() == true)
			{
				this.LastOpenPath = openFileDialog.FileName.Replace(openFileDialog.SafeFileName, "");
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text in fileNames)
				{
					this.OpenSpecFiles(text);
				}
			}
		}

		// Token: 0x060003CA RID: 970 RVA: 0x00011964 File Offset: 0x0000FB64
		public void OpenSubInsertIndFunctionFile()
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			if (string.IsNullOrEmpty(this.LastOpenPath))
			{
				openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			}
			else
			{
				openFileDialog.InitialDirectory = this.LastOpenPath;
			}
			openFileDialog.Filter = "Spec Files (*.tzf)|*.tzf;|All Files (*.*)|*.*";
			if (openFileDialog.ShowDialog() == true)
			{
				this.LastOpenPath = openFileDialog.FileName.Replace(openFileDialog.SafeFileName, "");
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text in fileNames)
				{
					this.OpenSpecFiles(text);
				}
			}
		}

		// Token: 0x060003CB RID: 971 RVA: 0x00011A20 File Offset: 0x0000FC20
		public void OpenReportTemplate()
		{
			OpenFileDialog openFileDialog = new OpenFileDialog();
			openFileDialog.Multiselect = true;
			openFileDialog.InitialDirectory = this.CurrentSetting.Connection.Workspace;
			openFileDialog.Filter = "Genero Report Files (*.4rp)|*.4rp";
			if (openFileDialog.ShowDialog() == true)
			{
				string[] fileNames = openFileDialog.FileNames;
				foreach (string text in fileNames)
				{
					string text2 = string.Format("\"{0}\"", text);
					Process.Start(text2);
				}
			}
		}

		// Token: 0x17000100 RID: 256
		// (get) Token: 0x060003CC RID: 972 RVA: 0x00011AB2 File Offset: 0x0000FCB2
		public Version Version
		{
			get
			{
				if (this._version == null)
				{
					this._version = Assembly.GetEntryAssembly().GetName().Version;
				}
				return this._version;
			}
		}

		// Token: 0x17000101 RID: 257
		// (get) Token: 0x060003CD RID: 973 RVA: 0x00011ADD File Offset: 0x0000FCDD
		// (set) Token: 0x060003CE RID: 974 RVA: 0x00011AE5 File Offset: 0x0000FCE5
		public string ErpVer
		{
			get
			{
				return this._erpVer;
			}
			set
			{
				this._erpVer = value;
			}
		}

		// Token: 0x060003CF RID: 975 RVA: 0x00011AF0 File Offset: 0x0000FCF0
		public void OpenSpecFiles(string file)
		{
			try
			{
				TzpManager tzpManager = new TzpManager(file);
				if (this.tzpMap.ContainsKey(tzpManager.ProgramKey))
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_SameFileNameAlreadyOpened") as string, tzpManager.ProgramName), Application.Current.FindResource("menu_FileOpen") as string);
				}
				else
				{
					this.tzpMap.Add(tzpManager.ProgramKey, tzpManager);
					tzpManager.LoadTzpFinished();
					EventAggregatorManager.Global.GetEvent<TzpFileLoaded>().Publish(tzpManager.ProgramKey);
				}
			}
			catch (VersionIncompatibleException ex)
			{
				string text = Application.Current.FindResource("Message_VersionInCompatible") as string;
				text = string.Format(text, file, ex.Remote, ex.Current);
				DesignerMessageBox.Show(text);
			}
			catch (NotInCurrentWorkspaceException ex2)
			{
				XElement settingNodeByWorkspace = this.SiteManager.GetSettingNodeByWorkspace(file);
				if (settingNodeByWorkspace != null)
				{
					string text2 = null;
					string value = settingNodeByWorkspace.Element("Connection").Element("Version").Value;
					string value2 = settingNodeByWorkspace.Attribute("uid").Value;
					try
					{
						text2 = RegistryReader.GetExeDir(value);
					}
					catch
					{
						string text3 = Application.Current.FindResource("Message_NotInstallCurrentVersionSpecDesigner") as string;
						text3 = string.Format(text3, value);
						DesignerMessageBox.Show(text3);
					}
					this.SiteManager.StartNewSpecDesigner(text2, value2, file);
				}
				else
				{
					DesignerMessageBox.Show(ex2.Message);
				}
			}
			catch (FileNotFoundException ex3)
			{
				DesignerMessageBox.Show(ex3.Message, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Hand);
			}
		}

		// Token: 0x060003D0 RID: 976 RVA: 0x00011CC8 File Offset: 0x0000FEC8
		private void CloseFile(PackageKey key)
		{
			if (!this.tzpMap.ContainsKey(key))
			{
				return;
			}
			this.tzpMap[key].ActionDefaults = null;
			this.tzpMap.Remove(key);
			this.undoRedoManagerMap.Remove(key);
		}

		// Token: 0x060003D1 RID: 977 RVA: 0x00011D05 File Offset: 0x0000FF05
		public void SaveSetting(PackageKey key)
		{
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Publish(key);
		}

		// Token: 0x1400000F RID: 15
		// (add) Token: 0x060003D2 RID: 978 RVA: 0x00011D18 File Offset: 0x0000FF18
		// (remove) Token: 0x060003D3 RID: 979 RVA: 0x00011D50 File Offset: 0x0000FF50
		public event SettingManager.LoadCommonDataProgressHandler ProgressChanged;

		// Token: 0x17000102 RID: 258
		// (get) Token: 0x060003D4 RID: 980 RVA: 0x00011D85 File Offset: 0x0000FF85
		// (set) Token: 0x060003D5 RID: 981 RVA: 0x00011D8D File Offset: 0x0000FF8D
		public bool CommonDataIsLoaded { get; private set; }

		// Token: 0x060003D6 RID: 982 RVA: 0x00011E1C File Offset: 0x0001001C
		public void LoadCommonData()
		{
			string text = string.Empty;
			text = this.CurrentSetting.Connection.Workspace;
			this.CommonDataIsLoaded = false;
			string text2 = Path.Combine(text, "mta", "ver");
			if (!File.Exists(text2))
			{
				throw new SettingFileNotFoundException(Application.Current.FindResource("Message_BasicDataVersionNotFound") as string);
			}
			string text3 = string.Empty;
			using (StreamReader streamReader = new StreamReader(text2))
			{
				text3 = streamReader.ReadLine();
			}
			if (string.IsNullOrWhiteSpace(text3))
			{
				throw new NotSupportedException(Application.Current.FindResource("Message_BasicDataVersionNotFound") as string);
			}
			if (text3 != string.Format("{0}.{1}", this.Version.Major, this.Version.Minor))
			{
				string text4 = string.Format(Application.Current.FindResource("Message_BasicDataVersionNotMatch") as string, new Version(text3), this.Version.Major, this.Version.MajorRevision);
				throw new VersionIncompatibleException(text4);
			}
			string text5 = Path.Combine(text, "mta", "about.xml");
			if (File.Exists(text5))
			{
				using (StreamReader streamReader2 = new StreamReader(text5))
				{
					XElement xelement = XElement.Parse(streamReader2.ReadToEnd());
					IEnumerable<XElement> enumerable = from query in xelement.Descendants("ErpVer")
						select (query);
					foreach (XElement xelement2 in enumerable)
					{
						this.ErpVer = xelement2.Attribute("value").Value;
					}
					goto IL_01D8;
				}
			}
			this.ErpVer = "1.0";
			IL_01D8:
			int num = 0;
			foreach (string text6 in this.specReferFiles)
			{
				try
				{
					string text7 = Path.Combine(text, text6);
					if (File.Exists(text7))
					{
						StreamReader streamReader3 = new StreamReader(text7);
						string fileName;
						switch (fileName = Path.GetFileName(text6))
						{
						case "core-br.spec":
							this.Info_CoreBr = streamReader3.ReadToEnd();
							break;
						case "mod-fd.spec":
							this.Info_ModFd = new ModFdInfo(streamReader3.ReadToEnd());
							break;
						case "tiptop.4ad":
							this.Info_ActionDefaults = streamReader3.ReadToEnd();
							this.ActionDefaults = XElement.Parse(this.Info_ActionDefaults);
							break;
						case "subroutines.xml":
							this.Info_Subroutines = streamReader3.ReadToEnd();
							break;
						case "checks.xml":
							this.Info_Checks = streamReader3.ReadToEnd();
							break;
						case "datatypes.xml":
							this.Info_DataTypes = XElement.Parse(streamReader3.ReadToEnd());
							break;
						case "items.xml":
							this.Info_Items = streamReader3.ReadToEnd();
							break;
						case "libraries.xml":
							this.Info_Libraries = streamReader3.ReadToEnd();
							break;
						case "messages.xml":
							this.Info_Messages = streamReader3.ReadToEnd();
							break;
						case "prog_rel.xml":
							this.Info_ProgRel = streamReader3.ReadToEnd();
							break;
						case "tables.xml":
							TableColumnHelper.Parse(text7, text);
							break;
						case "4gl.xml":
							this.Info_FGLKeyword = streamReader3.ReadToEnd();
							break;
						case "zooms.xml":
							this.Info_Zooms = streamReader3.ReadToEnd();
							break;
						case "top_global.inc":
							this.Info_Globals = streamReader3.ReadToEnd();
							break;
						case "tsd.xsd":
							this.Info_TsdXsd = streamReader3.ReadToEnd();
							break;
						case "code_sample.xml":
							this.Info_CodeSample = streamReader3.ReadToEnd();
							break;
						case "subinfo.xml":
							this.Info_SubInfo = streamReader3.ReadToEnd();
							break;
						case "top_global.xml":
							this.TopGlobals = XElement.Parse(streamReader3.ReadToEnd());
							break;
						case "languages.xml":
							this.Info_Languages = streamReader3.ReadToEnd();
							break;
						}
						Thread.Sleep(20);
						num++;
						if (this.ProgressChanged != null)
						{
							this.ProgressChanged(num, this.specReferFiles.Count<string>(), text6);
						}
						streamReader3.Close();
					}
				}
				catch
				{
					Application.Current.Dispatcher.Invoke(new Action(delegate
					{
						using (AutoCloseDialog autoCloseDialog = new AutoCloseDialog())
						{
							autoCloseDialog.Title = Application.Current.FindResource("Message_Warning") as string;
							autoCloseDialog.Message = Application.Current.FindResource("Message_BasicDataNotComplete") as string;
							autoCloseDialog.ImprovedShowDialog();
						}
					}), new object[0]);
				}
			}
			this.CommonDataIsLoaded = true;
			Application.Current.Dispatcher.BeginInvoke(new Action(delegate
			{
				EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Publish(null);
			}), new object[0]);
		}

		// Token: 0x060003D7 RID: 983 RVA: 0x00012430 File Offset: 0x00010630
		public bool CheckCommonData()
		{
			foreach (string text in this.specReferFiles)
			{
				string text2 = Path.Combine(this.CurrentSetting.Connection.Workspace, text);
				if (!File.Exists(text2))
				{
					return false;
				}
			}
			return true;
		}

		// Token: 0x060003D8 RID: 984 RVA: 0x00012484 File Offset: 0x00010684
		public string LoadToolBar(string classType)
		{
			if (string.IsNullOrEmpty(classType))
			{
				return null;
			}
			string text = "toolbar_" + classType.ToLowerInvariant() + ".4tb";
			string text2 = Path.Combine(this.CurrentSetting.Connection.Workspace, "4tb", text);
			if (File.Exists(text2))
			{
				StreamReader streamReader = new StreamReader(text2);
				return streamReader.ReadToEnd();
			}
			DesignerMessageBox.Show("Could not find the toolbar file:" + text2);
			return null;
		}

		// Token: 0x060003D9 RID: 985 RVA: 0x000124F8 File Offset: 0x000106F8
		public XElement LoadTableColumns(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			try
			{
				if (xelement != null)
				{
					XElement xelement2 = xelement.Elements().First<XElement>();
					if (xelement2.Attribute("name").Value != "")
					{
						XElement xelement3 = new XElement(xelement2);
						foreach (XAttribute xattribute in xelement3.Attributes())
						{
							xattribute.Value = "";
						}
						xelement.AddFirst(xelement3);
					}
				}
			}
			catch (NullReferenceException)
			{
				return null;
			}
			return xelement;
		}

		// Token: 0x060003DA RID: 986 RVA: 0x000125AC File Offset: 0x000107AC
		public string GetWorkspacePath()
		{
			return this.CurrentSetting.Connection.Workspace;
		}

		// Token: 0x17000103 RID: 259
		// (get) Token: 0x060003DB RID: 987 RVA: 0x000125F0 File Offset: 0x000107F0
		public List<DBOptions> ForDBs
		{
			get
			{
				if (this._forDBs == null)
				{
					this._forDBs = new List<DBOptions>();
					XElement xelement = XElement.Parse(this.Info_SubInfo);
					try
					{
						IEnumerable<DBOptions> enumerable = from db in xelement.Elements("for_db")
							select new DBOptions(db.Attribute("value").Value, db.Attribute("desc").Value);
						foreach (DBOptions dboptions in enumerable)
						{
							this._forDBs.Add(dboptions);
						}
					}
					catch
					{
					}
				}
				return this._forDBs;
			}
		}

		// Token: 0x17000104 RID: 260
		// (get) Token: 0x060003DC RID: 988 RVA: 0x000126DC File Offset: 0x000108DC
		public List<DimensionOption> Dimensions
		{
			get
			{
				if (this._dimensions == null)
				{
					this._dimensions = new List<DimensionOption>();
					XElement xelement = XElement.Parse(this.Info_SubInfo);
					IEnumerable<XElement> enumerable = from d in xelement.Elements("dimension")
						select (d);
					foreach (XElement xelement2 in enumerable)
					{
						DimensionOption dimensionOption = new DimensionOption(xelement2.Attribute("no").Value, xelement2.Attribute("desc").Value);
						IEnumerable<DimensionClassOption> enumerable2 = from c in xelement2.Elements("class")
							select new DimensionClassOption(c.Attribute("name").Value, c.Attribute("desc").Value);
						foreach (DimensionClassOption dimensionClassOption in enumerable2)
						{
							dimensionOption.Class.Add(dimensionClassOption);
						}
						this._dimensions.Add(dimensionOption);
					}
				}
				return this._dimensions;
			}
		}

		// Token: 0x17000105 RID: 261
		// (get) Token: 0x060003DD RID: 989 RVA: 0x00012838 File Offset: 0x00010A38
		public TOPSTDSettingModel TopstdSetting
		{
			get
			{
				if (this._topstdSetting == null)
				{
					this._topstdSetting = new TOPSTDSettingModel();
				}
				return this._topstdSetting;
			}
		}

		// Token: 0x04000152 RID: 338
		public string[] specReferFiles;

		// Token: 0x04000153 RID: 339
		private static SettingManager settingManager;

		// Token: 0x04000154 RID: 340
		public Dictionary<PackageKey, TzpManager> tzpMap = new Dictionary<PackageKey, TzpManager>(new PackageKey.PackageKeyComparer());

		// Token: 0x04000155 RID: 341
		public Dictionary<PackageKey, UndoRedoManager> undoRedoManagerMap = new Dictionary<PackageKey, UndoRedoManager>(new PackageKey.PackageKeyComparer());

		// Token: 0x04000156 RID: 342
		private SiteManager _siteManager;

		// Token: 0x04000157 RID: 343
		private FileSystemWatcher _fileSystemWatcher;

		// Token: 0x04000158 RID: 344
		private readonly string TTzip = "TT.zip";

		// Token: 0x0400015A RID: 346
		private string _lastOpenPath;

		// Token: 0x0400015B RID: 347
		private Version _version;

		// Token: 0x0400015C RID: 348
		private string _erpVer;

		// Token: 0x0400015D RID: 349
		public string Info_Checks;

		// Token: 0x0400015E RID: 350
		public XElement Info_DataTypes;

		// Token: 0x0400015F RID: 351
		public string Info_Items;

		// Token: 0x04000160 RID: 352
		public string Info_Libraries;

		// Token: 0x04000161 RID: 353
		public string Info_Subroutines;

		// Token: 0x04000162 RID: 354
		public string Info_Messages;

		// Token: 0x04000163 RID: 355
		public string Info_ProgRel;

		// Token: 0x04000164 RID: 356
		public string Info_Tables;

		// Token: 0x04000165 RID: 357
		public string Info_Zooms;

		// Token: 0x04000166 RID: 358
		public string Info_CoreBr;

		// Token: 0x04000167 RID: 359
		public ModFdInfo Info_ModFd;

		// Token: 0x04000168 RID: 360
		public string Info_ActionDefaults;

		// Token: 0x04000169 RID: 361
		public XElement ActionDefaults;

		// Token: 0x0400016A RID: 362
		public string Info_FGLKeyword;

		// Token: 0x0400016B RID: 363
		public string Info_Designer;

		// Token: 0x0400016C RID: 364
		public string Info_Globals;

		// Token: 0x0400016D RID: 365
		public string Info_TsdXsd;

		// Token: 0x0400016E RID: 366
		public string Info_CodeSample;

		// Token: 0x0400016F RID: 367
		public string Info_SubInfo;

		// Token: 0x04000170 RID: 368
		public XElement TopGlobals;

		// Token: 0x04000171 RID: 369
		public string Info_Languages;

		// Token: 0x04000173 RID: 371
		private List<DBOptions> _forDBs;

		// Token: 0x04000174 RID: 372
		private List<DimensionOption> _dimensions;

		// Token: 0x04000175 RID: 373
		private TOPSTDSettingModel _topstdSetting;

		// Token: 0x02000065 RID: 101
		// (Invoke) Token: 0x060003E8 RID: 1000
		public delegate void LoadCommonDataProgressHandler(int now, int count, string file);
	}
}
