using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using Microsoft.Practices.Prism.Events;
using SpecDesigner.Commands;
using SpecDesigner.Output;
using SpecDesigner.Search;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Logger;
using SpecDesignerCommon.TblUpdate;
using SpecDesignerPreference;
using Xceed.Wpf.AvalonDock;
using Xceed.Wpf.AvalonDock.Layout;
using Xceed.Wpf.AvalonDock.Layout.Serialization;

namespace SpecDesigner.Main
{
	// Token: 0x02000019 RID: 25
	public partial class SpecDesignerMainWindow : Window, INotifyPropertyChanged
	{
		// Token: 0x14000004 RID: 4
		// (add) Token: 0x060001F6 RID: 502 RVA: 0x00008530 File Offset: 0x00006730
		// (remove) Token: 0x060001F7 RID: 503 RVA: 0x00008568 File Offset: 0x00006768
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060001F8 RID: 504 RVA: 0x0000859D File Offset: 0x0000679D
		private void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x17000091 RID: 145
		// (get) Token: 0x060001F9 RID: 505 RVA: 0x000085B9 File Offset: 0x000067B9
		// (set) Token: 0x060001FA RID: 506 RVA: 0x000085C1 File Offset: 0x000067C1
		public bool CommonDataIsLoaded
		{
			get
			{
				return this._commonDataIsLoaded;
			}
			set
			{
				if (value == this._commonDataIsLoaded)
				{
					return;
				}
				this._commonDataIsLoaded = value;
				this.OnPropertyChanged("CommonDataIsLoaded");
			}
		}

		// Token: 0x060001FB RID: 507 RVA: 0x000085E0 File Offset: 0x000067E0
		public SpecDesignerMainWindow()
		{
			this.InitializeComponent();
			this.LoadLayoutSetting();
			this.BindMenuCommands(this);
			if (!SettingManager.Get().CommonDataIsLoaded)
			{
				this.CommonDataIsLoaded = false;
				SettingManager.Get().CommonDataLoaded += this.SpecDesignerMainWindow_CommonDataLoaded;
				MenuCommands.BasicDataUpdateCommand.Execute(null, this);
			}
			if (SettingManager.Get().ErpVer == "1.0")
			{
				this.simpleFormUpload.Visibility = Visibility.Collapsed;
			}
			this.preferenceMenu.ItemsSource = PreferenceManager.Current.Settings.CommonUsedSetting.EnableList;
			base.Closing += this.SpecDesignerMainWindow_Closing;
			base.DataContext = EditorWorkspace.This;
			base.GotFocus += this.SpecDesignerMainWindow_GotFocus;
			base.Activated += this.SpecDesignerMainWindow_Activated;
			base.Loaded += this.SpecDesignerMainWindow_Loaded;
			Application.Current.Resources.Add("AvalonDock_ThemeAero_BaseColor8", new SolidColorBrush(Color.FromRgb(208, 217, 246)));
			EventAggregatorManager.Global.GetEvent<ShowSearchBoxEvent>().Subscribe(new Action<string>(this.Subscribe_ShowSearchBox));
			EventAggregatorManager.Global.GetEvent<ShowReplaceBoxEvent>().Subscribe(new Action<string>(this.Subscribe_ShowReplaceBox));
			EventAggregatorManager.Global.GetEvent<ConnectionStatusChangedEvent>().Subscribe(new Action<ConnectionStatusChangedEventArgs>(this.Subscribe_ConnectionStatusChanged), ThreadOption.UIThread);
			ConnectionInfo.This.Initialize();
			this.worker = new BackgroundWorker();
			this.worker.DoWork += this.ready_to_getUpdateList;
			this.worker.RunWorkerCompleted += this.updateLocalRecord;
			this.worker.RunWorkerAsync();
			this.ShowLiteLayout();
		}

		// Token: 0x060001FC RID: 508 RVA: 0x000087B7 File Offset: 0x000069B7
		private void SpecDesignerMainWindow_CommonDataLoaded(object sender, EventArgs e)
		{
			SettingManager.Get().CommonDataLoaded -= this.SpecDesignerMainWindow_CommonDataLoaded;
			this.CommonDataIsLoaded = true;
		}

		// Token: 0x060001FD RID: 509 RVA: 0x000087D6 File Offset: 0x000069D6
		private void ready_to_getUpdateList(object sender, DoWorkEventArgs e)
		{
			this.um = new TblUpdateManager(SettingManager.Get().CurrentSetting.Connection.Workspace, Path.Combine(SettingManager.Get().CurrentSetting.Connection.UpdateURL, this.TBLUpdatedFile));
		}

		// Token: 0x060001FE RID: 510 RVA: 0x00008818 File Offset: 0x00006A18
		private void updateLocalRecord(object sender, RunWorkerCompletedEventArgs e)
		{
			this.worker.DoWork -= this.ready_to_getUpdateList;
			this.worker.RunWorkerCompleted -= this.updateLocalRecord;
			foreach (string text in this.um.GetNeddUpdateItems())
			{
				if (!string.IsNullOrEmpty(text))
				{
					TableColumnHelper.RemoveTableOrColumn(text);
				}
			}
			this.um.Dispose();
		}

		// Token: 0x060001FF RID: 511 RVA: 0x000088AC File Offset: 0x00006AAC
		private bool FormIsExists(string program, TzpType type)
		{
			return EditorWorkspace.This.IsExists(program, type);
		}

		// Token: 0x06000200 RID: 512 RVA: 0x000088BA File Offset: 0x00006ABA
		public void Subscribe_ShowSearchBox(string option)
		{
			SearchBoxControl.Show(this.dockManager, option);
		}

		// Token: 0x06000201 RID: 513 RVA: 0x000088C8 File Offset: 0x00006AC8
		public void Subscribe_ShowReplaceBox(string option)
		{
			ReplaceBoxControl.Show(this.dockManager, option);
		}

		// Token: 0x06000202 RID: 514 RVA: 0x000088D6 File Offset: 0x00006AD6
		public void Subscribe_ConnectionStatusChanged(ConnectionStatusChangedEventArgs args)
		{
			this.UpdateTitleWithConnectionStatus();
		}

		// Token: 0x06000203 RID: 515 RVA: 0x000088E0 File Offset: 0x00006AE0
		private void UpdateTitleWithConnectionStatus()
		{
			string text = Application.Current.FindResource("WinTitle_MainTitle") as string;
			string text2 = Application.Current.FindResource("Message_NotConnect") as string;
			if (ConnectionInfo.This.IsLogin)
			{
				text2 = Application.Current.FindResource("Message_ConnectInfo") as string;
				text2 = string.Format(text2, ConnectionInfo.This.LoginUser, ConnectionInfo.This.Area, ConnectionInfo.This.ServerIP);
			}
			base.Title = string.Format(text, SettingManager.Get().Version, SettingManager.Get().CurrentSetting.Name, text2);
			WelcomeViewModel.This.CurrentPlatformName = SettingManager.Get().CurrentSetting.Name;
			WelcomeViewModel welcomeViewModel = new WelcomeViewModel();
			welcomeViewModel.ChangePlatform();
		}

		// Token: 0x06000204 RID: 516 RVA: 0x000089A8 File Offset: 0x00006BA8
		private void SpecDesignerMainWindow_Loaded(object sender, RoutedEventArgs e)
		{
			this.UpdateTitleWithConnectionStatus();
			if (AppDomain.CurrentDomain.ActivationContext != null && AppDomain.CurrentDomain.SetupInformation.ActivationArguments.ActivationData != null)
			{
				string[] activationData = AppDomain.CurrentDomain.SetupInformation.ActivationArguments.ActivationData;
				if (activationData == null || activationData[0].EndsWith("application"))
				{
					return;
				}
				Uri uri = new Uri(activationData[0]);
				string localPath = uri.LocalPath;
				SettingManager.Get().OpenSpecFiles(localPath);
			}
			base.Loaded -= this.SpecDesignerMainWindow_Loaded;
		}

		// Token: 0x06000205 RID: 517 RVA: 0x00008A32 File Offset: 0x00006C32
		private void SpecDesignerMainWindow_Activated(object sender, EventArgs e)
		{
			if (this.focusedElement != null)
			{
				this.focusedElement.Focus();
			}
		}

		// Token: 0x06000206 RID: 518 RVA: 0x00008A48 File Offset: 0x00006C48
		private void SpecDesignerMainWindow_GotFocus(object sender, RoutedEventArgs e)
		{
			this.focusedElement = e.OriginalSource as UIElement;
		}

		// Token: 0x06000207 RID: 519 RVA: 0x00008A5B File Offset: 0x00006C5B
		private void SpecDesignerMainWindow_Closing(object sender, CancelEventArgs e)
		{
			if (!EditorWorkspace.This.CheckUnsavedFile())
			{
				e.Cancel = true;
				return;
			}
			this.SaveLayoutSetting();
			base.Closing -= this.SpecDesignerMainWindow_Closing;
			ConnectionManager.CloseTelnetChannel();
			Application.Current.Shutdown();
		}

		// Token: 0x06000208 RID: 520 RVA: 0x00008AB4 File Offset: 0x00006CB4
		public void ShowSelectedLayout(string contentID)
		{
			if (string.IsNullOrWhiteSpace(contentID))
			{
				return;
			}
			LayoutAnchorable layoutAnchorable = (from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
				where contentID == lc.ContentId
				select lc).FirstOrDefault<LayoutAnchorable>();
			if (layoutAnchorable != null)
			{
				if (layoutAnchorable.IsHidden)
				{
					this.dockManager.Layout.RemoveChild(layoutAnchorable);
					AnchorableShowStrategy anchorableShowStrategy = AnchorableShowStrategy.Left;
					string contentID2;
					switch (contentID2 = contentID)
					{
					case "SpecPropertyEditorLayout":
					case "FunctionListLayout":
					case "FormStructureLayout":
					case "WelcomeLayout":
					case "DatabaseSourceLayout":
						anchorableShowStrategy = AnchorableShowStrategy.Left;
						break;
					case "SpecErrorsLayout":
					case "SpecEditorLayout":
					case "BookmarksLayout":
						anchorableShowStrategy = AnchorableShowStrategy.Bottom;
						break;
					}
					layoutAnchorable.AddToLayout(this.dockManager, anchorableShowStrategy);
					layoutAnchorable.IsVisible = true;
					layoutAnchorable.IsSelected = true;
					return;
				}
				this.dockManager.Layout.RemoveChild(layoutAnchorable);
				layoutAnchorable.IsVisible = false;
				layoutAnchorable.IsSelected = false;
			}
		}

		// Token: 0x06000209 RID: 521 RVA: 0x00008C98 File Offset: 0x00006E98
		public void ShowLiteLayout()
		{
			if (PreferenceManager.Current.Settings.StandardView)
			{
				if (EditorWorkspace.This.FunctionListLayout_IsChecked)
				{
					(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
						where "FunctionListLayout" == lc.ContentId
						select lc).FirstOrDefault<LayoutAnchorable>().Show();
				}
				if (EditorWorkspace.This.SpecEditorLayout_IsChecked)
				{
					(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
						where "SpecEditorLayout" == lc.ContentId
						select lc).FirstOrDefault<LayoutAnchorable>().Show();
				}
				if (EditorWorkspace.This.BookmarksLayout_IsChecked)
				{
					(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
						where "BookmarksLayout" == lc.ContentId
						select lc).FirstOrDefault<LayoutAnchorable>().Show();
					return;
				}
			}
			else
			{
				(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
					where "FunctionListLayout" == lc.ContentId
					select lc).FirstOrDefault<LayoutAnchorable>().Hide(true);
				(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
					where "SpecEditorLayout" == lc.ContentId
					select lc).FirstOrDefault<LayoutAnchorable>().Hide(true);
				(from lc in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
					where "BookmarksLayout" == lc.ContentId
					select lc).FirstOrDefault<LayoutAnchorable>().Hide(true);
			}
		}

		// Token: 0x0600020A RID: 522 RVA: 0x00008EA4 File Offset: 0x000070A4
		private void LoadLayoutSetting()
		{
			try
			{
				Dictionary<string, string> dictionary = new Dictionary<string, string>();
				foreach (LayoutAnchorable layoutAnchorable in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>())
				{
					dictionary.Add(layoutAnchorable.ContentId, layoutAnchorable.Title);
				}
				if (File.Exists(SettingFileHelper.FullPathForLayoutSetting))
				{
					HashSet<LayoutAnchorable> hashSet = new HashSet<LayoutAnchorable>();
					int num = this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>().Count<LayoutAnchorable>();
					StringWriter stringWriter = new StringWriter();
					XmlLayoutSerializer xmlLayoutSerializer = new XmlLayoutSerializer(this.dockManager);
					xmlLayoutSerializer.Serialize(stringWriter);
					XmlLayoutSerializer xmlLayoutSerializer2 = new XmlLayoutSerializer(this.dockManager);
					xmlLayoutSerializer2.Deserialize(SettingFileHelper.FullPathForLayoutSetting);
					foreach (LayoutAnchorable layoutAnchorable2 in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>())
					{
						layoutAnchorable2.Title = dictionary[layoutAnchorable2.ContentId];
						PropertyInfo property = EditorWorkspace.This.GetType().GetProperty(layoutAnchorable2.ContentId + "_IsChecked");
						property.SetValue(EditorWorkspace.This, layoutAnchorable2.IsVisible, null);
						layoutAnchorable2.Hiding += this.Layout_Hiding;
						hashSet.Add(layoutAnchorable2);
					}
					if (num != this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>().Count<LayoutAnchorable>())
					{
						XmlLayoutSerializer xmlLayoutSerializer3 = new XmlLayoutSerializer(this.dockManager);
						StringReader stringReader = new StringReader(stringWriter.ToString());
						xmlLayoutSerializer3.Deserialize(stringReader);
						IEnumerable<LayoutAnchorable> enumerable = this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>();
						foreach (LayoutAnchorable layoutAnchorable3 in enumerable)
						{
							layoutAnchorable3.Hiding += this.Layout_Hiding;
						}
						LayoutAnchorable la;
						foreach (LayoutAnchorable layoutAnchorable4 in hashSet)
						{
							la = layoutAnchorable4;
							if (enumerable.Any<LayoutAnchorable>((LayoutAnchorable x) => x.Title == la.Title))
							{
								LayoutAnchorable layoutAnchorable5 = enumerable.First<LayoutAnchorable>((LayoutAnchorable x) => x.Title == la.Title);
								if (la.IsFloating)
								{
									layoutAnchorable5.FloatingHeight = la.FloatingHeight;
									layoutAnchorable5.FloatingLeft = la.FloatingLeft;
									layoutAnchorable5.FloatingTop = la.FloatingTop;
									layoutAnchorable5.FloatingWidth = la.FloatingWidth;
									layoutAnchorable5.Float();
								}
								if (la.IsHidden)
								{
									layoutAnchorable5.Hide(true);
								}
							}
						}
					}
				}
			}
			catch (Exception ex)
			{
				DSCLogger.Write(ex.Message, "LoadLayoutSetting");
			}
		}

		// Token: 0x0600020B RID: 523 RVA: 0x00009260 File Offset: 0x00007460
		private void SaveLayoutSetting()
		{
			try
			{
				string text = string.Empty;
				using (StringWriter stringWriter = new StringWriter())
				{
					LayoutAnchorable layoutAnchorable = (from l in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
						where l.ContentId == SearchBoxControl.ContentID
						select l).FirstOrDefault<LayoutAnchorable>();
					if (layoutAnchorable != null)
					{
						this.dockManager.Layout.RemoveChild(layoutAnchorable);
					}
					LayoutAnchorable layoutAnchorable2 = (from l in this.dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
						where l.ContentId == ReplaceBoxControl.ContentID
						select l).FirstOrDefault<LayoutAnchorable>();
					if (layoutAnchorable2 != null)
					{
						this.dockManager.Layout.RemoveChild(layoutAnchorable2);
					}
					XmlLayoutSerializer xmlLayoutSerializer = new XmlLayoutSerializer(this.dockManager);
					xmlLayoutSerializer.Serialize(stringWriter);
					text = stringWriter.ToString();
				}
				File.WriteAllText(SettingFileHelper.FullPathForLayoutSetting, text);
			}
			catch (Exception ex)
			{
				DSCLogger.Write(ex.Message, "SaveLayoutSetting");
			}
		}

		// Token: 0x0600020C RID: 524 RVA: 0x00009384 File Offset: 0x00007584
		private void BindMenuCommands(SpecDesignerMainWindow win)
		{
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FileOpenCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileOpen), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileOpen)));
			win.CommandBindings.Add(new CommandBinding(ApplicationCommands.Save, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileSave), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileSave)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FileCloseCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileClose), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileClose)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FileCloseAllCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileCloseAll), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileClose)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.DiffCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedDiff), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteDiff)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.InsertIndFunctionSetupCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedInsertIndFunctionSetup), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteInsertIndFunctionSetup)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ExitCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedExit), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteExit)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecificationDownloadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecificationDownload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecificationDownload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecThisVersionModifyCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecThisVersionModify), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecThisVersionModify)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SimpleFormSetupCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSimpleFormSetup), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSimpleFormSetup)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SimpleFormDownloadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSimpleFormDownload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSimpleFormDownload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SimpleFormUploadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSimpleFormUpload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSimpleFormUpload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecificationViewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecificationView), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecificationView)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.UncitedSpecificationViewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedUncitedSpecificationView), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteUncitedSpecificationView)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramDownloadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramDownload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramDownload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ExportDocxCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedExportDocx), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteExportDocx)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramThisVersionModifyCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramThisVersionModify), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramThisVersionModify)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramTestCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramTest), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramTest)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramDebugCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramDebug), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramDebug)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramErrorCheckCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramErrorCheck), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramErrorCheck)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowAboutCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowAbout), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowAbout)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowShorcutListCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowShorcutList), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowShorcutList)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowViCommandListCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowViCommandList), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowViCommandList)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.BasicDataUpdateCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedBasicDataUpdate), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteBasicDataUpdate)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SelectBasicDataUpdateCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSelectBasicDataUpdate), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSelectBasicDataUpdate)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.BaseDataViewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedBaseDataView), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteBaseDataView)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.TableViewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedTableView), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteTableView)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ColumnViewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedColumnView), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteColumnView)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.RegenerateBasicDataCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedReGenerateBasicData), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteReGenerateBasicData)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowLogCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowLog), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowLog)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ChangeLayoutCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedChangeLayout), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteChangeLayout)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FindCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFind), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFind)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ReplaceCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedReplace), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFind)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SiteManagerCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSiteManager)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowUncitedCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowUncited), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowUncited)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowSectionUncitedCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowSectionUncited), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowSectionUncited)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowCustomAdpListCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedCustomAdpList), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteCustomAdpList)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecificationUploadResetTAPCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecificationUploadResetTAP), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecificationUploadResetTAP)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SetFreeStyleCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSetFreeStyle), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSetFreeStyle)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ChangeProgramTemplateCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedChangeProgramTemplate), new CanExecuteRoutedEventHandler(MenuCommands.CanChangeProgramTemplate)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi900Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi900), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi900)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi901Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi901), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi901)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi910Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi910), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi910)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi920Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi920), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi920)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi600Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi600), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi600)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi650Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi650), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi650)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi140Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi140), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi140)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi150Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi150), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi150)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp168Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp168), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzp168)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp165Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp165), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzp165)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi210Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi210), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi210)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi220Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi220), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi220)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzq255Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzq255), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzq255)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp188Command, new ExecutedRoutedEventHandler(MenuCommands.Executedaadzp188)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi301Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi301)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi300Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi300)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp600Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp600)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.PreferenceWindowCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedPreferenceWindow), new CanExecuteRoutedEventHandler(MenuCommands.CanExecutePreferenceWindow)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SRRelationSettingCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSRRelationSetting), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSRRelationSetting)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ReportDownloadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedReportDownload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteReportDownload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ReportUploadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedReportUpload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteReportUpload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ReportOpenCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedReportOpen), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteReportOpen)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp270Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp270), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzp270)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.PreferenceWindowCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedPreferenceWindow), new CanExecuteRoutedEventHandler(MenuCommands.CanExecutePreferenceWindow)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ViewProgramInfomationCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedViewProgramInfomation), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteViewProgramInfomation)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp990ExpCommand, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp990Exp), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzp990Exp)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzp990ImpCommand, new ExecutedRoutedEventHandler(MenuCommands.Executedadzp990Imp), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzp990Imp)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ChangeStartargCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedChangeStartarg), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteChangeStartarg)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.azzi909Command, new ExecutedRoutedEventHandler(MenuCommands.Executedazzi909), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteazzi909)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.PreferenceShortcut, new ExecutedRoutedEventHandler(MenuCommands.ExecutedPreferenceShortcut)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecCodeUploadCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecCodeUpload), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecCodeUpload)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecCodeUploadAndRC3Command, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecCodeUploadAndRC3), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecCodeUploadAndRC3)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.PrecompileCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedPrecompile), new CanExecuteRoutedEventHandler(MenuCommands.CanExecutePrecompile)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ProgramRegenCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedProgramRegen), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteProgramRegen)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FormPreviewCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFormPreview), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFormPreview)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowGeneroUserGuideCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowGeneroUserGuide), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowGeneroUserGuide)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowTOPSTDLoginCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowTOPSTDLogin), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowTOPSTDLogin)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ToggleLayoutCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedToggleLayout), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteToggleLayout)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ToggleDiffEditableCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedToggleDiffEditable), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteToggleDiffEditable)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ViewFormDiffListCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedViewFormDiffList), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteViewFormDiffList)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SearchKeySettingWindowCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSearchKeySettingWindow), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSearchKeySettingWindow)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ShowServiceCloudLoginCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedShowServiceCloudLogin), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteShowServiceCloudLogin)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.UploadIndFunctionSetupCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedUploadIndFunctionSetup), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteUploaIndFunctionSetup)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzq001Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzq001), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzq001)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzq003Command, new ExecutedRoutedEventHandler(MenuCommands.Executedadzq003), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzq003)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FileOpenSpecCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileOpenSpec), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileOpenSpec)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.FileOpenCodeCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedFileOpenCode), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteFileOpenCode)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecAdjustContainerBlankAreaCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecAdjustContainerBlankArea), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecAdjustContainerBlankArea)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecOpenCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecOpen), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecOpen)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.SpecCheckInCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedSpecCheckIn), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSpecCheckIn)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.ToggleLiteCommand, new ExecutedRoutedEventHandler(MenuCommands.ExecutedToggleLite), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteToggleLite)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi888ExpCommand, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi888Exp), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi888Exp)));
			win.CommandBindings.Add(new CommandBinding(MenuCommands.adzi888ImpCommand, new ExecutedRoutedEventHandler(MenuCommands.Executedadzi888Imp), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteadzi888Imp)));
		}

		// Token: 0x0600020D RID: 525 RVA: 0x0000A400 File Offset: 0x00008600
		private void Layout_Hiding(object sender, CancelEventArgs e)
		{
			LayoutAnchorable layoutAnchorable = (LayoutAnchorable)sender;
			PropertyInfo property = EditorWorkspace.This.GetType().GetProperty(layoutAnchorable.ContentId + "_IsChecked");
			property.SetValue(EditorWorkspace.This, false, null);
		}

		// Token: 0x040000C5 RID: 197
		private bool _commonDataIsLoaded = true;

		// Token: 0x040000C6 RID: 198
		private readonly string TBLUpdatedFile = "AlterTableList.xml";

		// Token: 0x040000C7 RID: 199
		private BackgroundWorker worker;

		// Token: 0x040000C8 RID: 200
		private UIElement focusedElement;

		// Token: 0x040000C9 RID: 201
		private TblUpdateManager um;
	}
}
