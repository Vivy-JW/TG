using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Forms;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using ICSharpCode.SharpZipLib.Zip;
using Microsoft.Win32;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Logger;
using SpecDesignerCommon.Site.ViewModels;
using Xceed.Wpf.Toolkit;

namespace SpecDesignerCommon.Site
{
	// Token: 0x02000101 RID: 257
	public partial class SiteManager : Window, INotifyPropertyChanged
	{
		// Token: 0x1700026E RID: 622
		// (get) Token: 0x060008DF RID: 2271 RVA: 0x0002AFB1 File Offset: 0x000291B1
		// (set) Token: 0x060008E0 RID: 2272 RVA: 0x0002AFB9 File Offset: 0x000291B9
		public bool IsBusying
		{
			get
			{
				return this._isBusing;
			}
			set
			{
				this._isBusing = value;
				this.NotifyPropertyChanged("IsBusying");
			}
		}

		// Token: 0x060008E1 RID: 2273 RVA: 0x0002AFD0 File Offset: 0x000291D0
		public SiteManager()
		{
			this.InitializeComponent();
			this.LoadCommands();
			string text = global::System.Windows.Application.Current.FindResource("Site_SiteManagerTitle") as string;
			base.Title = string.Format(text, this.currentVersion);
			this.IsBusying = false;
			base.Closing += this.SiteManager_Closing;
			this.LoadSettings();
			base.Loaded += this.SiteManager_Loaded;
		}

		// Token: 0x060008E2 RID: 2274 RVA: 0x0002B064 File Offset: 0x00029264
		private void LoadCommands()
		{
			base.CommandBindings.Add(new CommandBinding(SiteManagerCommand.ExportCommand, new ExecutedRoutedEventHandler(this.ExecutedExport), new CanExecuteRoutedEventHandler(this.CanExecuteExport)));
			base.CommandBindings.Add(new CommandBinding(SiteManagerCommand.ImportCommand, new ExecutedRoutedEventHandler(this.ExecutedImport), new CanExecuteRoutedEventHandler(this.CanExecuteImport)));
		}

		// Token: 0x060008E3 RID: 2275 RVA: 0x0002B0D0 File Offset: 0x000292D0
		private void SiteManager_Loaded(object sender, RoutedEventArgs e)
		{
			SettingModelBase firstSetting = SiteManager._settingSource.FirstSetting;
			if (firstSetting != null)
			{
				firstSetting.IsSelected = true;
			}
			this.settingTV.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x060008E4 RID: 2276 RVA: 0x0002B104 File Offset: 0x00029304
		private void SiteManager_Closing(object sender, CancelEventArgs e)
		{
			if (this.SelectedSetting != null && !this.SelectedSetting.Connection.IsValidate)
			{
				this.SelectedSetting = null;
			}
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x060008E5 RID: 2277 RVA: 0x0002B1B4 File Offset: 0x000293B4
		private void CheckVersion()
		{
			if (!(this._selectedTreeItem is SettingModel))
			{
				return;
			}
			BackgroundWorker backgroundWorker = new BackgroundWorker();
			backgroundWorker.DoWork += delegate(object s, DoWorkEventArgs e)
			{
				ConnectionSetting connection = (this._selectedTreeItem as SettingModel).Connection;
				connection.SetRemoveVersionChecked(true);
				global::System.Windows.Application.Current.Properties["SiteManagerUID"] = connection.Setting.UID;
				UpdateManager.This.IsForceUpdate = this.IsNeedClosed;
				UpdateManager.This.checkFinished += this.CheckFinished;
				UpdateManager.This.AppCastIP = connection.UpdateURL;
			};
			backgroundWorker.RunWorkerAsync();
			this.IsBusying = true;
		}

		// Token: 0x060008E6 RID: 2278 RVA: 0x0002B25C File Offset: 0x0002945C
		private void CheckFinished(object sender, CheckVersionArgs e)
		{
			this.IsBusying = false;
			UpdateManager.This.checkFinished -= this.CheckFinished;
			ConnectionSetting selectedConnection = (this._selectedTreeItem as SettingModel).Connection;
			bool flag = false;
			string text = string.Format("{0}\\debug", selectedConnection.Workspace);
			if (File.Exists(text))
			{
				flag = true;
			}
			if (e.HasError)
			{
				if (e.ExceptionMessage is FileNotFoundException)
				{
					string text2 = global::System.Windows.Application.Current.FindResource("Message_VersionNotFound") as string;
					if (flag)
					{
						text2 = string.Format("{0}\n(From：'{1}')", text2, UpdateManager.This.AppCastIP);
						DSCLogger.Write(text2, string.Empty);
					}
					DesignerMessageBox.Show(text2, global::System.Windows.Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
				}
				else
				{
					string text3 = string.Format("{0}{1}{2}", global::System.Windows.Application.Current.FindResource("Message_GetVersionButError") as string, Environment.NewLine, e.ExceptionMessage);
					if (flag)
					{
						text3 = string.Format("{0}\n(From：'{1}')", text3, UpdateManager.This.AppCastIP);
						DSCLogger.Write(text3, string.Empty);
					}
					DesignerMessageBox.Show(text3, global::System.Windows.Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
				}
				MessageBoxResult messageBoxResult = DesignerMessageBox.Show(global::System.Windows.Application.Current.FindResource("Message_ContinueAndIgnoreConnectionError") as string, global::System.Windows.Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.YesNo, MessageBoxImage.Asterisk);
				if (messageBoxResult != MessageBoxResult.Yes)
				{
					return;
				}
			}
			base.Dispatcher.BeginInvoke(new Action(delegate
			{
				if (!string.IsNullOrEmpty(UpdateManager.This.Version))
				{
					selectedConnection.Version = UpdateManager.This.Version;
				}
				if (this.IsNeedClosed)
				{
					this.DialogResult = new bool?(true);
				}
				this.IsNeedClosed = true;
			}), new object[0]);
		}

		// Token: 0x060008E7 RID: 2279 RVA: 0x0002B403 File Offset: 0x00029603
		public SettingModel GetSettingByGuid(string guid)
		{
			return SiteManager._settingSource.GetSettingByGuid(guid);
		}

		// Token: 0x060008E8 RID: 2280 RVA: 0x0002B410 File Offset: 0x00029610
		private void LoadSettings()
		{
			SiteManager._settingSource = new SiteManagerModel(SiteFileHelper.Open());
			base.DataContext = SiteManager._settingSource;
		}

		// Token: 0x1700026F RID: 623
		// (get) Token: 0x060008E9 RID: 2281 RVA: 0x0002B433 File Offset: 0x00029633
		public static ICommand ConfirmCommand
		{
			get
			{
				if (SiteManager._confirmCommand == null)
				{
					SiteManager._confirmCommand = new RelayCommand(delegate(object p)
					{
						SiteManager.ExecuteConfirmConnection();
					});
				}
				return SiteManager._confirmCommand;
			}
		}

		// Token: 0x060008EA RID: 2282 RVA: 0x0002B468 File Offset: 0x00029668
		public static void ExecuteConfirmConnection()
		{
			XElement xelement = SiteManager._settingSource.ToXML();
			SiteFileHelper.Save(xelement);
		}

		// Token: 0x17000270 RID: 624
		// (get) Token: 0x060008EB RID: 2283 RVA: 0x0002B490 File Offset: 0x00029690
		public ICommand CancelCommand
		{
			get
			{
				if (SiteManager._cancelCommand == null)
				{
					SiteManager._cancelCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteCancelSetting();
					});
				}
				return SiteManager._cancelCommand;
			}
		}

		// Token: 0x060008EC RID: 2284 RVA: 0x0002B4C6 File Offset: 0x000296C6
		public void ExecuteCancelSetting()
		{
			this.SelectedSetting = null;
			base.Close();
		}

		// Token: 0x17000271 RID: 625
		// (get) Token: 0x060008ED RID: 2285 RVA: 0x0002B4E5 File Offset: 0x000296E5
		public RelayCommand ConnectCommand
		{
			get
			{
				if (SiteManager._connectCommand == null)
				{
					SiteManager._connectCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteConnect();
					}, (object p) => this.CanExecuteConnect());
				}
				return SiteManager._connectCommand;
			}
		}

		// Token: 0x060008EE RID: 2286 RVA: 0x0002B515 File Offset: 0x00029715
		public bool CanExecuteConnect()
		{
			this.SelectedSetting = this._selectedTreeItem as SettingModel;
			return this.SelectedSetting != null && this.SelectedSetting.Connection.IsInstalled && SiteManager.IsValid(this);
		}

		// Token: 0x060008EF RID: 2287 RVA: 0x0002B54C File Offset: 0x0002974C
		public static bool IsValid(DependencyObject parent)
		{
			bool flag = true;
			LocalValueEnumerator localValueEnumerator = parent.GetLocalValueEnumerator();
			while (localValueEnumerator.MoveNext())
			{
				LocalValueEntry localValueEntry = localValueEnumerator.Current;
				if (BindingOperations.IsDataBound(parent, localValueEntry.Property))
				{
					BindingOperations.GetBinding(parent, localValueEntry.Property);
					BindingExpression bindingExpression = BindingOperations.GetBindingExpression(parent, localValueEntry.Property);
					bindingExpression.UpdateSource();
					if (bindingExpression.HasError)
					{
						flag = false;
					}
				}
			}
			IEnumerable children = LogicalTreeHelper.GetChildren(parent);
			foreach (object obj in children)
			{
				if (obj is DependencyObject)
				{
					DependencyObject dependencyObject = (DependencyObject)obj;
					if (!SiteManager.IsValid(dependencyObject))
					{
						flag = false;
					}
				}
			}
			return flag;
		}

		// Token: 0x060008F0 RID: 2288 RVA: 0x0002B61C File Offset: 0x0002981C
		public void ExecuteConnect()
		{
			this.SelectedSetting = this._selectedTreeItem as SettingModel;
			if (!this.SelectedSetting.Connection.IsValidate)
			{
				DesignerMessageBox.Show(global::System.Windows.Application.Current.FindResource("Message_ConnectionIsNotValidate") as string);
				return;
			}
			try
			{
				RegistryReader.GetExeDir(this.SelectedSetting.Connection.Version);
			}
			catch (MissingFieldException)
			{
				DesignerMessageBox.Show(global::System.Windows.Application.Current.FindResource("Message_NotInstalled") as string);
				this.SelectedSetting = null;
				return;
			}
			if (this.conformToCurrent(this.SelectedSetting.Connection.Version))
			{
				SiteManager.ConfirmCommand.Execute(null);
				this.IsNeedClosed = true;
				this.CheckVersion();
				return;
			}
			SiteManager.ConfirmCommand.Execute(null);
			this.OpenSpecDesignerWithConnectInfo(this.SelectedSetting);
			this.SelectedSetting = null;
		}

		// Token: 0x060008F1 RID: 2289 RVA: 0x0002B704 File Offset: 0x00029904
		private void OnTreeNodeDoubleClick(object sender, MouseButtonEventArgs mouseEvtArgs)
		{
			TreeViewItem treeViewItem = sender as TreeViewItem;
			if (treeViewItem != null && treeViewItem.Header is SettingModel)
			{
				this.ConnectCommand.Execute(null);
			}
		}

		// Token: 0x060008F2 RID: 2290 RVA: 0x0002B734 File Offset: 0x00029934
		public void OpenSpecDesignerWithConnectInfo(SettingModel setting)
		{
			string exeDir = RegistryReader.GetExeDir(this.SelectedSetting.Connection.Version);
			Process.Start(new ProcessStartInfo(exeDir, this.SelectedSetting.UID));
		}

		// Token: 0x060008F3 RID: 2291 RVA: 0x0002B76E File Offset: 0x0002996E
		public void StartNewSpecDesigner(string exePath, string siteID, string tzp)
		{
			Process.Start(new ProcessStartInfo(exePath, string.Format(" \"{0}\" \"{1}\"", siteID, tzp)));
		}

		// Token: 0x14000021 RID: 33
		// (add) Token: 0x060008F4 RID: 2292 RVA: 0x0002B788 File Offset: 0x00029988
		// (remove) Token: 0x060008F5 RID: 2293 RVA: 0x0002B7C0 File Offset: 0x000299C0
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060008F6 RID: 2294 RVA: 0x0002B7F5 File Offset: 0x000299F5
		private void NotifyPropertyChanged(string info)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(info));
			}
		}

		// Token: 0x060008F7 RID: 2295 RVA: 0x0002B811 File Offset: 0x00029A11
		private void nameBox_LostKeyboardFocus(object sender, KeyboardFocusChangedEventArgs e)
		{
			this.UpdateTextBoxSource(sender as global::System.Windows.Controls.TextBox);
		}

		// Token: 0x060008F8 RID: 2296 RVA: 0x0002B820 File Offset: 0x00029A20
		private void UpdateTextBoxSource(global::System.Windows.Controls.TextBox nameTB)
		{
			BindingExpression bindingExpression = nameTB.GetBindingExpression(global::System.Windows.Controls.TextBox.TextProperty);
			if (bindingExpression != null)
			{
				bindingExpression.UpdateSource();
			}
		}

		// Token: 0x060008F9 RID: 2297 RVA: 0x0002B844 File Offset: 0x00029A44
		private void nameBox_IsVisibleChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if ((bool)e.NewValue)
			{
				global::System.Windows.Controls.TextBox textBox = sender as global::System.Windows.Controls.TextBox;
				Keyboard.Focus(sender as global::System.Windows.Controls.TextBox);
				textBox.SelectAll();
			}
		}

		// Token: 0x060008FA RID: 2298 RVA: 0x0002B878 File Offset: 0x00029A78
		private void dirButton_Click(object sender, RoutedEventArgs e)
		{
			FolderBrowserDialog folderBrowserDialog = new FolderBrowserDialog();
			if (Directory.Exists(this.workDirTB.Text))
			{
				folderBrowserDialog.SelectedPath = this.workDirTB.Text;
			}
			DialogResult dialogResult = folderBrowserDialog.ShowDialog();
			if (dialogResult == global::System.Windows.Forms.DialogResult.OK)
			{
				SettingModel settingModel = this._selectedTreeItem as SettingModel;
				if (settingModel != null)
				{
					settingModel.Connection.Workspace = folderBrowserDialog.SelectedPath;
				}
			}
		}

		// Token: 0x060008FB RID: 2299 RVA: 0x0002B8D9 File Offset: 0x00029AD9
		private void NameTextBox_KeyDown(object sender, global::System.Windows.Input.KeyEventArgs e)
		{
			if (e.Key == Key.Return)
			{
				this.UpdateTextBoxSource(sender as global::System.Windows.Controls.TextBox);
			}
		}

		// Token: 0x060008FC RID: 2300 RVA: 0x0002B8F0 File Offset: 0x00029AF0
		private void CheckNow_Click(object sender, RoutedEventArgs e)
		{
			this.IsBusying = true;
			this.IsNeedClosed = false;
			this.CheckVersion();
		}

		// Token: 0x060008FD RID: 2301 RVA: 0x0002B906 File Offset: 0x00029B06
		private void settingTV_SelectedItemChanged(object sender, RoutedPropertyChangedEventArgs<object> e)
		{
			this._selectedTreeItem = this.settingTV.SelectedItem as SettingModelBase;
		}

		// Token: 0x060008FE RID: 2302 RVA: 0x0002B920 File Offset: 0x00029B20
		private bool conformToCurrent(string ver)
		{
			Version version = new Version(ver);
			return this.currentVersion.Major == version.Major && this.currentVersion.Minor == version.Minor;
		}

		// Token: 0x060008FF RID: 2303 RVA: 0x0002B95C File Offset: 0x00029B5C
		public new SettingModel ShowDialog()
		{
			base.ShowDialog();
			return this.SelectedSetting;
		}

		// Token: 0x06000900 RID: 2304 RVA: 0x0002B96C File Offset: 0x00029B6C
		public SettingModel ShowDialog(string tzp)
		{
			string directoryName = Path.GetDirectoryName(tzp);
			SiteManager.SeekVersion(tzp);
			XElement settingNodeByWorkspace = this.GetSettingNodeByWorkspace(directoryName);
			if (settingNodeByWorkspace == null)
			{
				string text = global::System.Windows.Application.Current.FindResource("Message_TzpNotInWorkspace") as string;
				text = string.Format(text, tzp);
				DesignerMessageBox.Show(text);
				return null;
			}
			if (settingNodeByWorkspace != null)
			{
				string value = settingNodeByWorkspace.Element("Connection").Element("Version").Value;
				string text2 = null;
				try
				{
					text2 = RegistryReader.GetExeDir(value);
				}
				catch
				{
					string text3 = global::System.Windows.Application.Current.FindResource("Message_NotInstallCurrentVersionSpecDesigner") as string;
					text3 = string.Format(text3, value);
					DesignerMessageBox.Show(text3);
					return null;
				}
				Process.Start(new ProcessStartInfo(text2, string.Format(" {0} {1}", settingNodeByWorkspace.Attribute("uid").Value, tzp)));
				return null;
			}
			return null;
		}

		// Token: 0x06000901 RID: 2305 RVA: 0x0002BA64 File Offset: 0x00029C64
		public XElement GetSettingNodeByWorkspace(string dir)
		{
			if (!dir.EndsWith("\\"))
			{
				dir += "\\";
			}
			XElement xelement = SiteManager._settingSource.ToXML();
			IEnumerable<XElement> enumerable = xelement.Descendants("Connection");
			XElement xelement2 = null;
			foreach (XElement xelement3 in enumerable)
			{
				string text = xelement3.Element("Workspace").Value;
				if (!(text == ""))
				{
					if (!text.EndsWith("\\"))
					{
						text += "\\";
					}
					if (dir.StartsWith(text))
					{
						xelement2 = xelement3.Parent;
						break;
					}
				}
			}
			return xelement2;
		}

		// Token: 0x06000902 RID: 2306 RVA: 0x0002BB38 File Offset: 0x00029D38
		private static string SeekVersion(string tzpFile)
		{
			using (FileStream fileStream = new FileStream(tzpFile, FileMode.Open, FileAccess.Read))
			{
				using (ZipFile zipFile = new ZipFile(fileStream))
				{
					ZipEntry entry = zipFile.GetEntry("ver");
					if (entry != null)
					{
						StreamReader streamReader = new StreamReader(zipFile.GetInputStream(entry));
						return streamReader.ReadToEnd();
					}
				}
			}
			throw new Exception(global::System.Windows.Application.Current.FindResource("Message_VersionNotFoundInTZS") as string);
		}

		// Token: 0x06000903 RID: 2307 RVA: 0x0002BBCC File Offset: 0x00029DCC
		private void ExportSettings()
		{
			Microsoft.Win32.SaveFileDialog saveFileDialog = new Microsoft.Win32.SaveFileDialog();
			saveFileDialog.FileName = "sdsettings.xml";
			saveFileDialog.DefaultExt = ".xml";
			saveFileDialog.Filter = "XML documents (.xml)|*.xml";
			if (saveFileDialog.ShowDialog() == true)
			{
				string fileName = saveFileDialog.FileName;
				SiteManager._settingSource.ExportTo(fileName);
			}
		}

		// Token: 0x06000904 RID: 2308 RVA: 0x0002BC30 File Offset: 0x00029E30
		private void ImportSettings()
		{
			Microsoft.Win32.OpenFileDialog openFileDialog = new Microsoft.Win32.OpenFileDialog();
			openFileDialog.FileName = "sdsettings.xml";
			openFileDialog.DefaultExt = ".xml";
			openFileDialog.Filter = "XML documents (.xml)|*.xml";
			if (openFileDialog.ShowDialog() == true)
			{
				string fileName = openFileDialog.FileName;
				SiteManager._settingSource.ImportFrom(fileName);
			}
		}

		// Token: 0x06000905 RID: 2309 RVA: 0x0002BC94 File Offset: 0x00029E94
		public void CanExecuteExport(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000906 RID: 2310 RVA: 0x0002BC9D File Offset: 0x00029E9D
		public void ExecutedExport(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			this.ExportSettings();
		}

		// Token: 0x06000907 RID: 2311 RVA: 0x0002BCAC File Offset: 0x00029EAC
		public void CanExecuteImport(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000908 RID: 2312 RVA: 0x0002BCB5 File Offset: 0x00029EB5
		public void ExecutedImport(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			this.ImportSettings();
		}

		// Token: 0x06000909 RID: 2313 RVA: 0x0002BCC4 File Offset: 0x00029EC4
		private void ModifyButton_Click(object sender, RoutedEventArgs e)
		{
			if (MessageBoxResult.Yes == DesignerMessageBox.Show(global::System.Windows.Application.Current.FindResource("Site_ModifyPromptWarning") as string, global::System.Windows.Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation))
			{
				this.loginPromptBox.IsReadOnly = (this.passwordPrompBox.IsReadOnly = (this.areaPromptBox.IsReadOnly = false));
			}
		}

		// Token: 0x0600090D RID: 2317 RVA: 0x0002BEB4 File Offset: 0x0002A0B4
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		[EditorBrowsable(EditorBrowsableState.Never)]
		[DebuggerNonUserCode]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			if (connectionId == 2)
			{
				EventSetter eventSetter = new EventSetter();
				eventSetter.Event = global::System.Windows.Controls.Control.MouseDoubleClickEvent;
				eventSetter.Handler = new MouseButtonEventHandler(this.OnTreeNodeDoubleClick);
				((Style)target).Setters.Add(eventSetter);
				return;
			}
			if (connectionId != 6)
			{
				return;
			}
			((global::System.Windows.Controls.TextBox)target).LostKeyboardFocus += this.nameBox_LostKeyboardFocus;
			((global::System.Windows.Controls.TextBox)target).KeyDown += this.NameTextBox_KeyDown;
			((global::System.Windows.Controls.TextBox)target).IsVisibleChanged += this.nameBox_IsVisibleChanged;
		}

		// Token: 0x04000322 RID: 802
		public SettingModel SelectedSetting;

		// Token: 0x04000323 RID: 803
		private static SiteManagerModel _settingSource;

		// Token: 0x04000324 RID: 804
		private static RelayCommand _confirmCommand = null;

		// Token: 0x04000325 RID: 805
		private static RelayCommand _cancelCommand = null;

		// Token: 0x04000326 RID: 806
		private static RelayCommand _connectCommand;

		// Token: 0x04000327 RID: 807
		private SettingModelBase _selectedTreeItem;

		// Token: 0x04000328 RID: 808
		private Version currentVersion = Assembly.GetEntryAssembly().GetName().Version;

		// Token: 0x04000329 RID: 809
		private bool _isBusing;

		// Token: 0x0400032A RID: 810
		private bool IsNeedClosed = true;
	}
}
