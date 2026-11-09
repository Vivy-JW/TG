using System;
using System.Diagnostics;
using System.IO;
using System.Windows;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Export;
using SpecDesigner.Export.Adapters;
using SpecDesigner.FormEditor;
using SpecDesigner.FormEditor.Relation;
using SpecDesigner.FormEditor.Views;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesigner.Main;
using SpecDesigner.SpecEditor;
using SpecDesigner.SpecEditor.Views;
using SpecDesigner.TableViewer;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Site.ViewModels;
using SpecDesignerCommon.ViewModel;
using SpecDesignerCommon.Views;
using SpecDesignerPreference;

namespace SpecDesigner.Commands
{
	// Token: 0x02000012 RID: 18
	public class MenuCommands
	{
		// Token: 0x1700001D RID: 29
		// (get) Token: 0x0600008E RID: 142 RVA: 0x00003DD8 File Offset: 0x00001FD8
		public static RoutedCommand FileOpenCommand
		{
			get
			{
				if (MenuCommands._fileOpenCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.O, ModifierKeys.Control));
					MenuCommands._fileOpenCommand = new RoutedCommand("FileOpenCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._fileOpenCommand;
			}
		}

		// Token: 0x0600008F RID: 143 RVA: 0x00003E20 File Offset: 0x00002020
		public static void CanExecuteFileOpen(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x06000090 RID: 144 RVA: 0x00003E39 File Offset: 0x00002039
		public static void ExecutedFileOpen(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenFile();
		}

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x06000091 RID: 145 RVA: 0x00003E4C File Offset: 0x0000204C
		public static RoutedCommand SpecOpenCommand
		{
			get
			{
				if (MenuCommands._specOpenCommand == null)
				{
					MenuCommands._specOpenCommand = new RoutedCommand("SpecOpenCommand", typeof(MenuCommands));
				}
				return MenuCommands._specOpenCommand;
			}
		}

		// Token: 0x06000092 RID: 146 RVA: 0x00003E73 File Offset: 0x00002073
		public static void CanExecuteSpecOpen(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x06000093 RID: 147 RVA: 0x00003E8C File Offset: 0x0000208C
		public static void ExecutedSpecOpen(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenFile(1);
		}

		// Token: 0x06000094 RID: 148 RVA: 0x00003EA0 File Offset: 0x000020A0
		public static void CanExecuteFileSave(object sender, CanExecuteRoutedEventArgs e)
		{
			if (MenuCommands.ProgramKey == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000095 RID: 149 RVA: 0x00003EBE File Offset: 0x000020BE
		public static void ExecutedFileSave(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			SettingManager.Get().SaveSetting(MenuCommands.ProgramKey);
		}

		// Token: 0x1700001F RID: 31
		// (get) Token: 0x06000096 RID: 150 RVA: 0x00003EDC File Offset: 0x000020DC
		public static RoutedCommand FileCloseCommand
		{
			get
			{
				if (MenuCommands._fileCloseCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.W, ModifierKeys.Control));
					MenuCommands._fileCloseCommand = new RoutedCommand("FileCloseCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._fileCloseCommand;
			}
		}

		// Token: 0x06000097 RID: 151 RVA: 0x00003F24 File Offset: 0x00002124
		public static void CanExecuteFileClose(object sender, CanExecuteRoutedEventArgs e)
		{
			if (EditorWorkspace.This.ActiveDocument == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000098 RID: 152 RVA: 0x00003F41 File Offset: 0x00002141
		public static void ExecutedFileClose(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			EditorWorkspace.This.ActiveDocument.CloseCommand.Execute(null);
		}

		// Token: 0x17000020 RID: 32
		// (get) Token: 0x06000099 RID: 153 RVA: 0x00003F64 File Offset: 0x00002164
		public static RoutedCommand FileCloseAllCommand
		{
			get
			{
				if (MenuCommands._fileCloseAllCommand == null)
				{
					MenuCommands._fileCloseAllCommand = new RoutedCommand("FileCloseAllCommand", typeof(MenuCommands));
				}
				return MenuCommands._fileCloseAllCommand;
			}
		}

		// Token: 0x0600009A RID: 154 RVA: 0x00003F8B File Offset: 0x0000218B
		public static void ExecutedFileCloseAll(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			EditorWorkspace.This.CloseAll();
		}

		// Token: 0x17000021 RID: 33
		// (get) Token: 0x0600009B RID: 155 RVA: 0x00003FA4 File Offset: 0x000021A4
		public static RoutedCommand ExitCommand
		{
			get
			{
				if (MenuCommands._exitCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F4, ModifierKeys.Alt));
					MenuCommands._exitCommand = new RoutedCommand("ExitCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._exitCommand;
			}
		}

		// Token: 0x0600009C RID: 156 RVA: 0x00003FEC File Offset: 0x000021EC
		public static void CanExecuteExit(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600009D RID: 157 RVA: 0x00003FF8 File Offset: 0x000021F8
		public static void ExecutedExit(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			if (SettingManager.Get().tzpMap.Count > 0)
			{
				MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmExit") as string, Application.Current.FindResource("Message_ConfirmExitTitle") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation);
				if (messageBoxResult == MessageBoxResult.No)
				{
					return;
				}
			}
			Application.Current.Shutdown();
		}

		// Token: 0x17000022 RID: 34
		// (get) Token: 0x0600009E RID: 158 RVA: 0x00004063 File Offset: 0x00002263
		public static RoutedCommand DiffCommand
		{
			get
			{
				if (MenuCommands._diffCommand == null)
				{
					MenuCommands._diffCommand = new RoutedCommand("DiffCommand", typeof(MenuCommands));
				}
				return MenuCommands._diffCommand;
			}
		}

		// Token: 0x0600009F RID: 159 RVA: 0x0000408A File Offset: 0x0000228A
		public static void CanExecuteDiff(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x060000A0 RID: 160 RVA: 0x000040A3 File Offset: 0x000022A3
		public static void ExecutedDiff(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenDiffFile();
		}

		// Token: 0x17000023 RID: 35
		// (get) Token: 0x060000A1 RID: 161 RVA: 0x000040B8 File Offset: 0x000022B8
		public static RoutedCommand ShowAboutCommand
		{
			get
			{
				if (MenuCommands._showAboutCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F1, ModifierKeys.Control));
					MenuCommands._showAboutCommand = new RoutedCommand("ShowAboutCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._showAboutCommand;
			}
		}

		// Token: 0x060000A2 RID: 162 RVA: 0x00004100 File Offset: 0x00002300
		public static void CanExecuteShowAbout(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000A3 RID: 163 RVA: 0x0000410C File Offset: 0x0000230C
		public static void ExecutedShowAbout(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			InformationWindow informationWindow = new InformationWindow();
			informationWindow.Show();
		}

		// Token: 0x17000024 RID: 36
		// (get) Token: 0x060000A4 RID: 164 RVA: 0x0000412C File Offset: 0x0000232C
		public static RoutedCommand ShowShorcutListCommand
		{
			get
			{
				if (MenuCommands._showShorcutListCommand == null)
				{
					MenuCommands._showShorcutListCommand = new RoutedCommand("ShowShorcutListCommand", typeof(MenuCommands));
				}
				return MenuCommands._showShorcutListCommand;
			}
		}

		// Token: 0x060000A5 RID: 165 RVA: 0x00004153 File Offset: 0x00002353
		public static void CanExecuteShowShorcutList(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000A6 RID: 166 RVA: 0x0000415C File Offset: 0x0000235C
		public static void ExecutedShowShorcutList(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ShortcutListWindow.This.Show();
		}

		// Token: 0x17000025 RID: 37
		// (get) Token: 0x060000A7 RID: 167 RVA: 0x0000416F File Offset: 0x0000236F
		public static RoutedCommand ShowViCommandListCommand
		{
			get
			{
				if (MenuCommands._showViCommandListCommand == null)
				{
					MenuCommands._showViCommandListCommand = new RoutedCommand("ShowViCommandListCommand", typeof(MenuCommands));
				}
				return MenuCommands._showViCommandListCommand;
			}
		}

		// Token: 0x060000A8 RID: 168 RVA: 0x00004196 File Offset: 0x00002396
		public static void CanExecuteShowViCommandList(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000A9 RID: 169 RVA: 0x0000419F File Offset: 0x0000239F
		public static void ExecutedShowViCommandList(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ViCommandListWindow.This.Show();
		}

		// Token: 0x17000026 RID: 38
		// (get) Token: 0x060000AA RID: 170 RVA: 0x000041B4 File Offset: 0x000023B4
		public static RoutedCommand BaseDataViewCommand
		{
			get
			{
				if (MenuCommands._baseDataViewCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					MenuCommands._baseDataViewCommand = new RoutedCommand("BaseDataViewCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._baseDataViewCommand;
			}
		}

		// Token: 0x060000AB RID: 171 RVA: 0x000041ED File Offset: 0x000023ED
		public static void CanExecuteBaseDataView(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x060000AC RID: 172 RVA: 0x00004200 File Offset: 0x00002400
		public static void ExecutedBaseDataView(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			BaseDataViewWindow baseDataViewWindow = new BaseDataViewWindow();
			baseDataViewWindow.Show();
		}

		// Token: 0x17000027 RID: 39
		// (get) Token: 0x060000AD RID: 173 RVA: 0x00004220 File Offset: 0x00002420
		public static RoutedCommand TableViewCommand
		{
			get
			{
				if (MenuCommands._tableViewCommand == null)
				{
					MenuCommands._tableViewCommand = new RoutedCommand("TableViewCommand", typeof(MenuCommands));
				}
				return MenuCommands._tableViewCommand;
			}
		}

		// Token: 0x060000AE RID: 174 RVA: 0x00004247 File Offset: 0x00002447
		public static void CanExecuteTableView(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x060000AF RID: 175 RVA: 0x00004259 File Offset: 0x00002459
		public static void ExecutedTableView(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TableViewerWindow.This.Show();
		}

		// Token: 0x17000028 RID: 40
		// (get) Token: 0x060000B0 RID: 176 RVA: 0x0000426C File Offset: 0x0000246C
		public static RoutedCommand ColumnViewCommand
		{
			get
			{
				if (MenuCommands._columnViewCommand == null)
				{
					MenuCommands._columnViewCommand = new RoutedCommand("ColumnViewCommand", typeof(MenuCommands));
				}
				return MenuCommands._columnViewCommand;
			}
		}

		// Token: 0x060000B1 RID: 177 RVA: 0x00004293 File Offset: 0x00002493
		public static void CanExecuteColumnView(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x060000B2 RID: 178 RVA: 0x000042A5 File Offset: 0x000024A5
		public static void ExecutedColumnView(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ColumnViewerWindow.This.Show();
		}

		// Token: 0x17000029 RID: 41
		// (get) Token: 0x060000B3 RID: 179 RVA: 0x000042B8 File Offset: 0x000024B8
		public static RoutedCommand BasicDataUpdateCommand
		{
			get
			{
				if (MenuCommands._basicDataUpdateCommand == null)
				{
					MenuCommands._basicDataUpdateCommand = new RoutedCommand("BasicDataUpdateCommand", typeof(MenuCommands));
				}
				return MenuCommands._basicDataUpdateCommand;
			}
		}

		// Token: 0x060000B4 RID: 180 RVA: 0x000042DF File Offset: 0x000024DF
		public static void CanExecuteBasicDataUpdate(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000B5 RID: 181 RVA: 0x000042E8 File Offset: 0x000024E8
		public static void ExecutedBasicDataUpdate(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (ConnectionManager.BasicDataUpdate())
			{
				ProgressBarViewModel.Instance.Message = Application.Current.FindResource("Message_BasicDataUpdating").ToString();
				ProgressBar.Instance.ShowDialog();
			}
		}

		// Token: 0x1700002A RID: 42
		// (get) Token: 0x060000B6 RID: 182 RVA: 0x00004321 File Offset: 0x00002521
		public static RoutedCommand SelectBasicDataUpdateCommand
		{
			get
			{
				if (MenuCommands._selectbasicDataUpdateCommand == null)
				{
					MenuCommands._selectbasicDataUpdateCommand = new RoutedCommand("SelectBasicDataUpdateCommand", typeof(MenuCommands));
				}
				return MenuCommands._selectbasicDataUpdateCommand;
			}
		}

		// Token: 0x060000B7 RID: 183 RVA: 0x00004348 File Offset: 0x00002548
		public static void CanExecuteSelectBasicDataUpdate(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x060000B8 RID: 184 RVA: 0x0000435C File Offset: 0x0000255C
		public static void ExecutedSelectBasicDataUpdate(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			string text = "";
			if (!PreferenceManager.Current.Settings.StandardView)
			{
				text = "SIMPLE";
			}
			if (string.IsNullOrEmpty(text))
			{
				ConnectionManager.SelectBasicDataUpdate();
				return;
			}
			ConnectionManager.SelectBasicDataUpdate(text, (EditorWorkspace.This.ActiveDocument == null) ? "" : EditorWorkspace.This.ActiveDocument.Title);
		}

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x060000B9 RID: 185 RVA: 0x000043C5 File Offset: 0x000025C5
		public static RoutedCommand RegenerateBasicDataCommand
		{
			get
			{
				if (MenuCommands._regenerateBasicDataCommand == null)
				{
					MenuCommands._regenerateBasicDataCommand = new RoutedCommand("RegenerateBasicDataCommand", typeof(MenuCommands));
				}
				return MenuCommands._regenerateBasicDataCommand;
			}
		}

		// Token: 0x060000BA RID: 186 RVA: 0x000043EC File Offset: 0x000025EC
		public static void CanExecuteReGenerateBasicData(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000BB RID: 187 RVA: 0x000043F5 File Offset: 0x000025F5
		public static void ExecutedReGenerateBasicData(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.ReGenerateBasicData();
		}

		// Token: 0x1700002C RID: 44
		// (get) Token: 0x060000BC RID: 188 RVA: 0x00004403 File Offset: 0x00002603
		public static RoutedCommand SpecificationViewCommand
		{
			get
			{
				if (MenuCommands._specificationViewCommand == null)
				{
					MenuCommands._specificationViewCommand = new RoutedCommand("SpecificationViewCommand", typeof(MenuCommands));
				}
				return MenuCommands._specificationViewCommand;
			}
		}

		// Token: 0x060000BD RID: 189 RVA: 0x0000442C File Offset: 0x0000262C
		public static void CanExecuteSpecificationView(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (null == MenuCommands.ProgramKey || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x060000BE RID: 190 RVA: 0x00004488 File Offset: 0x00002688
		public static void ExecutedSpecificationView(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SpecificationViewWindow specificationViewWindow = new SpecificationViewWindow(MenuCommands.ProgramKey);
			specificationViewWindow.Show();
		}

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x060000BF RID: 191 RVA: 0x000044AD File Offset: 0x000026AD
		public static RoutedCommand UncitedSpecificationViewCommand
		{
			get
			{
				if (MenuCommands._uncitedSpecificationViewCommand == null)
				{
					MenuCommands._uncitedSpecificationViewCommand = new RoutedCommand("UncitedSpecificationViewCommand", typeof(MenuCommands));
				}
				return MenuCommands._uncitedSpecificationViewCommand;
			}
		}

		// Token: 0x060000C0 RID: 192 RVA: 0x000044D4 File Offset: 0x000026D4
		public static void CanExecuteUncitedSpecificationView(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (null == MenuCommands.ProgramKey || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo == null)
			{
				e.CanExecute = false;
				return;
			}
			if (!SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).IsStandardProgram)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C1 RID: 193 RVA: 0x0000454C File Offset: 0x0000274C
		public static void ExecutedUncitedSpecificationView(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SpecificationViewWindow specificationViewWindow = new SpecificationViewWindow(MenuCommands.ProgramKey, true);
			specificationViewWindow.Show();
		}

		// Token: 0x1700002E RID: 46
		// (get) Token: 0x060000C2 RID: 194 RVA: 0x00004572 File Offset: 0x00002772
		public static RoutedCommand SpecificationUploadResetTAPCommand
		{
			get
			{
				if (MenuCommands._specificationUploadResetTAPCommand == null)
				{
					MenuCommands._specificationUploadResetTAPCommand = new RoutedCommand("SpecificationUploadResetTAPCommand", typeof(MenuCommands));
				}
				return MenuCommands._specificationUploadResetTAPCommand;
			}
		}

		// Token: 0x060000C3 RID: 195 RVA: 0x0000459C File Offset: 0x0000279C
		public static void CanExecuteSpecificationUploadResetTAP(object sender, CanExecuteRoutedEventArgs e)
		{
			if (!(MenuCommands.ProgramKey != null) || (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type != TzpType.Code && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type != TzpType.ReportCode) || !("1" == ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).Ver))
			{
				e.CanExecute = false;
				return;
			}
			if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C4 RID: 196 RVA: 0x00004634 File Offset: 0x00002834
		public static void ExecutedSpecificationUploadResetTAP(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.ReBuildProgram(MenuCommands.ProgramKey.Program, ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).Ver, "CODE", ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).TAP.Attribute("type").Value, ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).TAP.Attribute("identity").Value);
		}

		// Token: 0x1700002F RID: 47
		// (get) Token: 0x060000C5 RID: 197 RVA: 0x000046C0 File Offset: 0x000028C0
		public static RoutedCommand SpecificationDownloadCommand
		{
			get
			{
				if (MenuCommands._specificationDownloadCommand == null)
				{
					MenuCommands._specificationDownloadCommand = new RoutedCommand("SpecificationDownloadCommand", typeof(MenuCommands));
				}
				return MenuCommands._specificationDownloadCommand;
			}
		}

		// Token: 0x060000C6 RID: 198 RVA: 0x000046E7 File Offset: 0x000028E7
		public static void CanExecuteSpecificationDownload(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000C7 RID: 199 RVA: 0x000046F0 File Offset: 0x000028F0
		public static void ExecutedSpecificationDownload(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.SpecificationDownload(SettingManager.Get().CurrentSetting.Connection.Workspace, !PreferenceManager.Current.Settings.StandardView);
		}

		// Token: 0x17000030 RID: 48
		// (get) Token: 0x060000C8 RID: 200 RVA: 0x00004725 File Offset: 0x00002925
		public static RoutedCommand ProgramDownloadCommand
		{
			get
			{
				if (MenuCommands._programDownloadCommand == null)
				{
					MenuCommands._programDownloadCommand = new RoutedCommand("ProgramDownloadCommand", typeof(MenuCommands));
				}
				return MenuCommands._programDownloadCommand;
			}
		}

		// Token: 0x060000C9 RID: 201 RVA: 0x0000474C File Offset: 0x0000294C
		public static void CanExecuteProgramDownload(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000CA RID: 202 RVA: 0x00004755 File Offset: 0x00002955
		public static void ExecutedProgramDownload(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.ProgramDownload(SettingManager.Get().CurrentSetting.Connection.Workspace);
		}

		// Token: 0x17000031 RID: 49
		// (get) Token: 0x060000CB RID: 203 RVA: 0x00004778 File Offset: 0x00002978
		public static RoutedCommand ProgramDebugCommand
		{
			get
			{
				if (MenuCommands._programDebugCommand == null)
				{
					MenuCommands._programDebugCommand = new RoutedCommand("ProgramDebugCommand", typeof(MenuCommands));
				}
				return MenuCommands._programDebugCommand;
			}
		}

		// Token: 0x060000CC RID: 204 RVA: 0x000047A0 File Offset: 0x000029A0
		public static void CanExecuteProgramDebug(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code)
			{
				flag = true;
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000CD RID: 205 RVA: 0x000047ED File Offset: 0x000029ED
		public static void ExecutedProgramDebug(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.ProgramDebug(MenuCommands.ProgramKey.Program);
		}

		// Token: 0x17000032 RID: 50
		// (get) Token: 0x060000CE RID: 206 RVA: 0x00004806 File Offset: 0x00002A06
		public static RoutedCommand ProgramErrorCheckCommand
		{
			get
			{
				if (MenuCommands._programErrorCheckCommand == null)
				{
					MenuCommands._programErrorCheckCommand = new RoutedCommand("ProgramErrorCheckCommand", typeof(MenuCommands));
				}
				return MenuCommands._programErrorCheckCommand;
			}
		}

		// Token: 0x060000CF RID: 207 RVA: 0x00004830 File Offset: 0x00002A30
		public static void CanExecuteProgramErrorCheck(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code)
			{
				flag = true;
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000D0 RID: 208 RVA: 0x0000487D File Offset: 0x00002A7D
		public static void ExecutedProgramErrorCheck(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			EventAggregatorManager.Global.GetEvent<ProgramErrorCheckEvent>().Publish(MenuCommands.ProgramKey);
		}

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x060000D1 RID: 209 RVA: 0x0000489A File Offset: 0x00002A9A
		public static RoutedCommand ProgramTestCommand
		{
			get
			{
				if (MenuCommands._programTestCommand == null)
				{
					MenuCommands._programTestCommand = new RoutedCommand("ProgramTestCommand", typeof(MenuCommands));
				}
				return MenuCommands._programTestCommand;
			}
		}

		// Token: 0x060000D2 RID: 210 RVA: 0x000048C4 File Offset: 0x00002AC4
		public static void CanExecuteProgramTest(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code)
			{
				flag = true;
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000D3 RID: 211 RVA: 0x00004911 File Offset: 0x00002B11
		public static void ExecutedProgramTest(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.ProgramTest(MenuCommands.ProgramKey.Program);
		}

		// Token: 0x17000034 RID: 52
		// (get) Token: 0x060000D4 RID: 212 RVA: 0x0000492A File Offset: 0x00002B2A
		public static RoutedCommand ProgramRegenCommand
		{
			get
			{
				if (MenuCommands._programRegenCommand == null)
				{
					MenuCommands._programRegenCommand = new RoutedCommand("ProgramRegenCommand", typeof(MenuCommands));
				}
				return MenuCommands._programRegenCommand;
			}
		}

		// Token: 0x060000D5 RID: 213 RVA: 0x00004954 File Offset: 0x00002B54
		public static void CanExecuteProgramRegen(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code)
			{
				ProgramInformation programInfo = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey);
				flag = programInfo != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000D6 RID: 214 RVA: 0x000049CC File Offset: 0x00002BCC
		public static void ExecutedProgramRegen(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (MessageBoxResult.OK == DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_RegenComfirm") as string, MenuCommands.ProgramKey.Program), Application.Current.FindResource("menu_ShowProgramRegen") as string, MessageBoxButton.OKCancel, MessageBoxImage.Asterisk))
			{
				ProgramInformation programInfo = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey);
				XAttribute xattribute = programInfo.TAP.Attribute("prog");
				XAttribute xattribute2 = programInfo.TAP.Attribute("type");
				XAttribute xattribute3 = programInfo.TAP.Attribute("module");
				string text = string.Format("adzp147 {0} {1} {2}", (xattribute != null) ? xattribute.Value : "", (xattribute2 != null) ? xattribute2.Value : "", (xattribute3 != null) ? xattribute3.Value : "");
				ConnectionManager.RunProgram(text);
			}
		}

		// Token: 0x17000035 RID: 53
		// (get) Token: 0x060000D7 RID: 215 RVA: 0x00004ABB File Offset: 0x00002CBB
		public static RoutedCommand ShowLogCommand
		{
			get
			{
				if (MenuCommands._showLogCommand == null)
				{
					MenuCommands._showLogCommand = new RoutedCommand("ShowLogCommand", typeof(MenuCommands));
				}
				return MenuCommands._showLogCommand;
			}
		}

		// Token: 0x060000D8 RID: 216 RVA: 0x00004AE2 File Offset: 0x00002CE2
		public static void CanExecuteShowLog(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000D9 RID: 217 RVA: 0x00004AEB File Offset: 0x00002CEB
		public static void ExecutedShowLog(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionInfo.This.CreateLogWindow();
		}

		// Token: 0x17000036 RID: 54
		// (get) Token: 0x060000DA RID: 218 RVA: 0x00004AFE File Offset: 0x00002CFE
		public static RoutedCommand ShowUncitedCommand
		{
			get
			{
				if (MenuCommands._showUncitedCommand == null)
				{
					MenuCommands._showUncitedCommand = new RoutedCommand("ShowUncitedCommand", typeof(MenuCommands));
				}
				return MenuCommands._showUncitedCommand;
			}
		}

		// Token: 0x060000DB RID: 219 RVA: 0x00004B28 File Offset: 0x00002D28
		public static void CanExecuteShowUncited(object sender, CanExecuteRoutedEventArgs e)
		{
			if (MenuCommands.ProgramKey != null)
			{
				e.CanExecute = !SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).IsStandardProgram && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060000DC RID: 220 RVA: 0x00004B80 File Offset: 0x00002D80
		public static void ExecutedShowUncited(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CustomAdpListWindow customAdpListWindow = new CustomAdpListWindow(CustomAdpListWindow.ContentType.cite);
			customAdpListWindow.ShowDialog();
		}

		// Token: 0x17000037 RID: 55
		// (get) Token: 0x060000DD RID: 221 RVA: 0x00004BA2 File Offset: 0x00002DA2
		public static RoutedCommand ShowSectionUncitedCommand
		{
			get
			{
				if (MenuCommands._showSectionUncitedCommand == null)
				{
					MenuCommands._showSectionUncitedCommand = new RoutedCommand("ShowSectionUncitedCommand", typeof(MenuCommands));
				}
				return MenuCommands._showSectionUncitedCommand;
			}
		}

		// Token: 0x060000DE RID: 222 RVA: 0x00004BC9 File Offset: 0x00002DC9
		public static void CanExecuteShowSectionUncited(object sender, CanExecuteRoutedEventArgs e)
		{
			if (MenuCommands.ProgramKey != null)
			{
				e.CanExecute = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060000DF RID: 223 RVA: 0x00004C00 File Offset: 0x00002E00
		public static void ExecutedShowSectionUncited(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CustomAdpListWindow customAdpListWindow = new CustomAdpListWindow(CustomAdpListWindow.ContentType.section);
			customAdpListWindow.ShowDialog();
		}

		// Token: 0x17000038 RID: 56
		// (get) Token: 0x060000E0 RID: 224 RVA: 0x00004C22 File Offset: 0x00002E22
		public static RoutedCommand ShowCustomAdpListCommand
		{
			get
			{
				if (MenuCommands._showCustomAdpListCommand == null)
				{
					MenuCommands._showCustomAdpListCommand = new RoutedCommand("ShowCustomAdpListCommand", typeof(MenuCommands));
				}
				return MenuCommands._showCustomAdpListCommand;
			}
		}

		// Token: 0x060000E1 RID: 225 RVA: 0x00004C49 File Offset: 0x00002E49
		public static void CanExecuteCustomAdpList(object sender, CanExecuteRoutedEventArgs e)
		{
			if (MenuCommands.ProgramKey != null)
			{
				e.CanExecute = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Code;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060000E2 RID: 226 RVA: 0x00004C80 File Offset: 0x00002E80
		public static void ExecutedCustomAdpList(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CustomAdpListWindow customAdpListWindow = new CustomAdpListWindow(CustomAdpListWindow.ContentType.src);
			customAdpListWindow.ShowDialog();
		}

		// Token: 0x17000039 RID: 57
		// (get) Token: 0x060000E3 RID: 227 RVA: 0x00004CA2 File Offset: 0x00002EA2
		public static RoutedCommand SiteManagerCommand
		{
			get
			{
				if (MenuCommands._siteManagerCommand == null)
				{
					MenuCommands._siteManagerCommand = new RoutedCommand("SiteManagerCommand", typeof(MenuCommands));
				}
				return MenuCommands._siteManagerCommand;
			}
		}

		// Token: 0x060000E4 RID: 228 RVA: 0x00004CCC File Offset: 0x00002ECC
		public static void ExecutedSiteManager(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingModel settingModel = SettingManager.Get().SiteManager.ShowDialog();
			ConnectionSetting connection = SettingManager.Get().CurrentSetting.Connection;
			if (settingModel != null && settingModel.Connection != connection)
			{
				SettingManager.Get().SiteManager.OpenSpecDesignerWithConnectInfo(settingModel);
			}
		}

		// Token: 0x1700003A RID: 58
		// (get) Token: 0x060000E5 RID: 229 RVA: 0x00004D21 File Offset: 0x00002F21
		public static RoutedCommand ChangeLayoutCommand
		{
			get
			{
				if (MenuCommands._changeLayoutCommand == null)
				{
					MenuCommands._changeLayoutCommand = new RoutedCommand("ChangeLayoutCommand", typeof(MenuCommands));
				}
				return MenuCommands._changeLayoutCommand;
			}
		}

		// Token: 0x060000E6 RID: 230 RVA: 0x00004D48 File Offset: 0x00002F48
		public static void CanExecuteChangeLayout(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000E7 RID: 231 RVA: 0x00004D51 File Offset: 0x00002F51
		public static void ExecutedChangeLayout(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
		}

		// Token: 0x1700003B RID: 59
		// (get) Token: 0x060000E8 RID: 232 RVA: 0x00004D5C File Offset: 0x00002F5C
		public static RoutedCommand FindCommand
		{
			get
			{
				if (MenuCommands._findCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F, ModifierKeys.Control));
					MenuCommands._findCommand = new RoutedCommand("FindCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._findCommand;
			}
		}

		// Token: 0x060000E9 RID: 233 RVA: 0x00004DA4 File Offset: 0x00002FA4
		public static void CanExecuteFind(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = MenuCommands.ProgramKey != null;
		}

		// Token: 0x060000EA RID: 234 RVA: 0x00004DB7 File Offset: 0x00002FB7
		public static void ExecutedFind(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			EventAggregatorManager.Global.GetEvent<ShowSearchBoxEvent>().Publish(EditorWorkspace.This.ActiveDocument.SelectionContent);
		}

		// Token: 0x1700003C RID: 60
		// (get) Token: 0x060000EB RID: 235 RVA: 0x00004DE0 File Offset: 0x00002FE0
		public static RoutedCommand ReplaceCommand
		{
			get
			{
				if (MenuCommands._replaceCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.H, ModifierKeys.Control));
					MenuCommands._replaceCommand = new RoutedCommand("ReplaceCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._replaceCommand;
			}
		}

		// Token: 0x060000EC RID: 236 RVA: 0x00004E28 File Offset: 0x00003028
		public static void ExecutedReplace(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			EventAggregatorManager.Global.GetEvent<ShowReplaceBoxEvent>().Publish(EditorWorkspace.This.ActiveDocument.SelectionContent);
		}

		// Token: 0x1700003D RID: 61
		// (get) Token: 0x060000ED RID: 237 RVA: 0x00004E50 File Offset: 0x00003050
		public static RoutedCommand ExportDocxCommand
		{
			get
			{
				if (MenuCommands._exportDocxCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.E, ModifierKeys.Control));
					MenuCommands._exportDocxCommand = new RoutedCommand("ExportDocxCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._exportDocxCommand;
			}
		}

		// Token: 0x060000EE RID: 238 RVA: 0x00004E98 File Offset: 0x00003098
		public static void ExecutedExportDocx(object sender, ExecutedRoutedEventArgs e)
		{
			FormViewModel formViewModel = EditorWorkspace.This.ActiveDocument as FormViewModel;
			FormEditorMainWindow formEditorMainWindow = formViewModel.UI as FormEditorMainWindow;
			if (formEditorMainWindow != null)
			{
				ExportSpecification exportSpecification = new ExportSpecification(new DocxExportAdapter(formEditorMainWindow));
				exportSpecification.Export(SettingManager.Get().CurrentSetting.Connection.Workspace);
			}
		}

		// Token: 0x060000EF RID: 239 RVA: 0x00004EEC File Offset: 0x000030EC
		public static void CanExecuteExportDocx(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			FormViewModel formViewModel = EditorWorkspace.This.ActiveDocument as FormViewModel;
			FormEditorMainWindow formEditorMainWindow = ((formViewModel == null) ? null : (formViewModel.UI as FormEditorMainWindow));
			e.CanExecute = formEditorMainWindow != null;
		}

		// Token: 0x1700003E RID: 62
		// (get) Token: 0x060000F0 RID: 240 RVA: 0x00004F44 File Offset: 0x00003144
		public static RoutedCommand SpecCodeUploadCommand
		{
			get
			{
				if (MenuCommands._specCodeUploadCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F6));
					MenuCommands._specCodeUploadCommand = new RoutedCommand("SpecCodeUploadCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._specCodeUploadCommand;
			}
		}

		// Token: 0x060000F1 RID: 241 RVA: 0x00004F8C File Offset: 0x0000318C
		public static void CanExecuteSpecCodeUpload(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null)
			{
				if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).isIndFun)
				{
					e.CanExecute = false;
					return;
				}
				if (e.Parameter != null && e.Parameter is TzpType)
				{
					TzpType? tzpType = new TzpType?((TzpType)e.Parameter);
					flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == tzpType && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
				}
				else
				{
					flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000F2 RID: 242 RVA: 0x00005084 File Offset: 0x00003284
		public static void ExecutedSpecCodeUpload(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			TzpType type = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type;
			if (type == TzpType.Form)
			{
				if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsTsdValidated && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsUIValidated)
				{
					MenuCommands.ConfirmSaveAndUpload(TzpType.Form);
					return;
				}
				MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmUploadTsdEvenValidationFail") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation);
				if (messageBoxResult == MessageBoxResult.Yes)
				{
					MenuCommands.ConfirmSaveAndUpload(TzpType.Form);
					return;
				}
			}
			else
			{
				MenuCommands.ConfirmSaveAndUpload(SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type);
			}
		}

		// Token: 0x1700003F RID: 63
		// (get) Token: 0x060000F3 RID: 243 RVA: 0x00005148 File Offset: 0x00003348
		public static RoutedCommand SpecCodeUploadAndRC3Command
		{
			get
			{
				if (MenuCommands._specCodeUploadAndRC3Command == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F7));
					MenuCommands._specCodeUploadAndRC3Command = new RoutedCommand("SpecCodeUploadAndRC3Command", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._specCodeUploadAndRC3Command;
			}
		}

		// Token: 0x060000F4 RID: 244 RVA: 0x00005190 File Offset: 0x00003390
		public static void CanExecuteSpecCodeUploadAndRC3(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null)
			{
				if (!SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type.Equals(TzpType.Form))
				{
					flag = false;
				}
				else if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "M" || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "S" || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "Q")
				{
					if (e.Parameter != null && e.Parameter is TzpType)
					{
						TzpType? tzpType = new TzpType?((TzpType)e.Parameter);
						flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == tzpType && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
					}
					else
					{
						flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
					}
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000F5 RID: 245 RVA: 0x000052F8 File Offset: 0x000034F8
		public static void ExecutedSpecCodeUploadAndRC3(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			TzpType type = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type;
			if (type == TzpType.Form)
			{
				if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsTsdValidated && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsUIValidated)
				{
					MenuCommands.ConfirmSaveAndUpload(TzpType.Form, true);
					return;
				}
				MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmUploadTsdEvenValidationFail") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation);
				if (messageBoxResult == MessageBoxResult.Yes)
				{
					MenuCommands.ConfirmSaveAndUpload(TzpType.Form, true);
					return;
				}
			}
			else
			{
				MenuCommands.ConfirmSaveAndUpload(SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type);
			}
		}

		// Token: 0x17000040 RID: 64
		// (get) Token: 0x060000F6 RID: 246 RVA: 0x000053C0 File Offset: 0x000035C0
		public static RoutedCommand PrecompileCommand
		{
			get
			{
				if (MenuCommands._precompileCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F6, ModifierKeys.Shift));
					MenuCommands._precompileCommand = new RoutedCommand("PrecompileCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._precompileCommand;
			}
		}

		// Token: 0x060000F7 RID: 247 RVA: 0x00005408 File Offset: 0x00003608
		public static void CanExecutePrecompile(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null)
			{
				if (e.Parameter != null && e.Parameter is TzpType)
				{
					TzpType? tzpType = new TzpType?((TzpType)e.Parameter);
					flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == tzpType && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).CanPrecompile;
				}
				else
				{
					flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).CanPrecompile;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000F8 RID: 248 RVA: 0x000054E0 File Offset: 0x000036E0
		public static void ExecutedPrecompile(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MenuCommands.ForceUpdateSource();
			switch (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type)
			{
			case TzpType.Code:
				MenuCommands.ConfirmSaveAndPreCompile(SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type);
				break;
			case TzpType.CodeSpec:
				break;
			case TzpType.Form:
			{
				if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsTsdValidated && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.IsUIValidated)
				{
					MenuCommands.ConfirmSaveAndPreCompile(TzpType.Form);
					return;
				}
				MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmUploadTsdEvenValidationFail") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation);
				if (messageBoxResult == MessageBoxResult.Yes)
				{
					MenuCommands.ConfirmSaveAndPreCompile(TzpType.Form);
					return;
				}
				break;
			}
			default:
				return;
			}
		}

		// Token: 0x17000041 RID: 65
		// (get) Token: 0x060000F9 RID: 249 RVA: 0x000055B5 File Offset: 0x000037B5
		public static RoutedCommand adzi140Command
		{
			get
			{
				if (MenuCommands._adzi140Command == null)
				{
					MenuCommands._adzi140Command = new RoutedCommand("adzi140Command", typeof(MenuCommands));
				}
				return MenuCommands._adzi140Command;
			}
		}

		// Token: 0x060000FA RID: 250 RVA: 0x000055DC File Offset: 0x000037DC
		public static void CanExecuteadzi140(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000FB RID: 251 RVA: 0x000055E5 File Offset: 0x000037E5
		public static void Executedadzi140(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi140");
		}

		// Token: 0x17000042 RID: 66
		// (get) Token: 0x060000FC RID: 252 RVA: 0x000055F2 File Offset: 0x000037F2
		public static RoutedCommand CommonUsedMenuCommand
		{
			get
			{
				if (MenuCommands._commonUsedMenuCommand == null)
				{
					MenuCommands._commonUsedMenuCommand = new RoutedCommand("CommonUsedMenuCommand", typeof(MenuCommands));
				}
				return MenuCommands._commonUsedMenuCommand;
			}
		}

		// Token: 0x060000FD RID: 253 RVA: 0x00005619 File Offset: 0x00003819
		public static void CanExecuteCommonUsedMenu(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060000FE RID: 254 RVA: 0x00005622 File Offset: 0x00003822
		public static void ExecuteCommonUsedMenu(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("");
		}

		// Token: 0x17000043 RID: 67
		// (get) Token: 0x060000FF RID: 255 RVA: 0x0000562F File Offset: 0x0000382F
		public static RoutedCommand adzi150Command
		{
			get
			{
				if (MenuCommands._adzi150Command == null)
				{
					MenuCommands._adzi150Command = new RoutedCommand("adzi150Command", typeof(MenuCommands));
				}
				return MenuCommands._adzi150Command;
			}
		}

		// Token: 0x06000100 RID: 256 RVA: 0x00005656 File Offset: 0x00003856
		public static void CanExecuteadzi150(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000101 RID: 257 RVA: 0x0000565F File Offset: 0x0000385F
		public static void Executedadzi150(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi150");
		}

		// Token: 0x17000044 RID: 68
		// (get) Token: 0x06000102 RID: 258 RVA: 0x0000566C File Offset: 0x0000386C
		public static RoutedCommand adzi210Command
		{
			get
			{
				if (MenuCommands._adzi210Command == null)
				{
					MenuCommands._adzi210Command = new RoutedCommand("adzi210Command", typeof(MenuCommands));
				}
				return MenuCommands._adzi210Command;
			}
		}

		// Token: 0x06000103 RID: 259 RVA: 0x00005693 File Offset: 0x00003893
		public static void CanExecuteadzi210(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000104 RID: 260 RVA: 0x0000569C File Offset: 0x0000389C
		public static void Executedadzi210(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi210");
		}

		// Token: 0x17000045 RID: 69
		// (get) Token: 0x06000105 RID: 261 RVA: 0x000056A9 File Offset: 0x000038A9
		public static RoutedCommand adzi220Command
		{
			get
			{
				if (MenuCommands._adzi220Command == null)
				{
					MenuCommands._adzi220Command = new RoutedCommand("adzi220Command", typeof(MenuCommands));
				}
				return MenuCommands._adzi220Command;
			}
		}

		// Token: 0x06000106 RID: 262 RVA: 0x000056D0 File Offset: 0x000038D0
		public static void CanExecuteadzi220(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000107 RID: 263 RVA: 0x000056D9 File Offset: 0x000038D9
		public static void Executedadzi220(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi220");
		}

		// Token: 0x17000046 RID: 70
		// (get) Token: 0x06000108 RID: 264 RVA: 0x000056E6 File Offset: 0x000038E6
		public static RoutedCommand adzi400Command
		{
			get
			{
				if (MenuCommands._adzi400Command == null)
				{
					MenuCommands._adzi400Command = new RoutedCommand("adzi400Command", typeof(MenuCommands));
				}
				return MenuCommands._adzi400Command;
			}
		}

		// Token: 0x06000109 RID: 265 RVA: 0x0000570D File Offset: 0x0000390D
		public static void CanExecuteadzi400(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600010A RID: 266 RVA: 0x00005716 File Offset: 0x00003916
		public static void Executedadzi400(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi400");
		}

		// Token: 0x17000047 RID: 71
		// (get) Token: 0x0600010B RID: 267 RVA: 0x00005723 File Offset: 0x00003923
		public static RoutedCommand adzq490Command
		{
			get
			{
				if (MenuCommands._adzq490Command == null)
				{
					MenuCommands._adzq490Command = new RoutedCommand("adzq490Command", typeof(MenuCommands));
				}
				return MenuCommands._adzq490Command;
			}
		}

		// Token: 0x0600010C RID: 268 RVA: 0x0000574A File Offset: 0x0000394A
		public static void CanExecuteadzq490(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600010D RID: 269 RVA: 0x00005753 File Offset: 0x00003953
		public static void Executedadzq490(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzq490");
		}

		// Token: 0x17000048 RID: 72
		// (get) Token: 0x0600010E RID: 270 RVA: 0x00005760 File Offset: 0x00003960
		public static RoutedCommand azzi600Command
		{
			get
			{
				if (MenuCommands._azzi600Command == null)
				{
					MenuCommands._azzi600Command = new RoutedCommand("azzi600Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi600Command;
			}
		}

		// Token: 0x0600010F RID: 271 RVA: 0x00005787 File Offset: 0x00003987
		public static void CanExecuteazzi600(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000110 RID: 272 RVA: 0x00005790 File Offset: 0x00003990
		public static void Executedazzi600(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi600");
		}

		// Token: 0x17000049 RID: 73
		// (get) Token: 0x06000111 RID: 273 RVA: 0x0000579D File Offset: 0x0000399D
		public static RoutedCommand azzi650Command
		{
			get
			{
				if (MenuCommands._azzi650Command == null)
				{
					MenuCommands._azzi650Command = new RoutedCommand("azzi650Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi650Command;
			}
		}

		// Token: 0x06000112 RID: 274 RVA: 0x000057C4 File Offset: 0x000039C4
		public static void CanExecuteazzi650(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000113 RID: 275 RVA: 0x000057CD File Offset: 0x000039CD
		public static void Executedazzi650(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi650");
		}

		// Token: 0x1700004A RID: 74
		// (get) Token: 0x06000114 RID: 276 RVA: 0x000057DA File Offset: 0x000039DA
		public static RoutedCommand azzi900Command
		{
			get
			{
				if (MenuCommands._azzi900Command == null)
				{
					MenuCommands._azzi900Command = new RoutedCommand("azzi900Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi900Command;
			}
		}

		// Token: 0x06000115 RID: 277 RVA: 0x00005801 File Offset: 0x00003A01
		public static void CanExecuteazzi900(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000116 RID: 278 RVA: 0x0000580A File Offset: 0x00003A0A
		public static void Executedazzi900(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi900");
		}

		// Token: 0x1700004B RID: 75
		// (get) Token: 0x06000117 RID: 279 RVA: 0x00005817 File Offset: 0x00003A17
		public static RoutedCommand azzi901Command
		{
			get
			{
				if (MenuCommands._azzi901Command == null)
				{
					MenuCommands._azzi901Command = new RoutedCommand("azzi901Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi901Command;
			}
		}

		// Token: 0x06000118 RID: 280 RVA: 0x0000583E File Offset: 0x00003A3E
		public static void CanExecuteazzi901(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000119 RID: 281 RVA: 0x00005847 File Offset: 0x00003A47
		public static void Executedazzi901(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi901");
		}

		// Token: 0x1700004C RID: 76
		// (get) Token: 0x0600011A RID: 282 RVA: 0x00005854 File Offset: 0x00003A54
		public static RoutedCommand azzi910Command
		{
			get
			{
				if (MenuCommands._azzi910Command == null)
				{
					MenuCommands._azzi910Command = new RoutedCommand("azzi910Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi910Command;
			}
		}

		// Token: 0x0600011B RID: 283 RVA: 0x0000587B File Offset: 0x00003A7B
		public static void CanExecuteazzi910(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600011C RID: 284 RVA: 0x00005884 File Offset: 0x00003A84
		public static void Executedazzi910(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi910");
		}

		// Token: 0x1700004D RID: 77
		// (get) Token: 0x0600011D RID: 285 RVA: 0x00005891 File Offset: 0x00003A91
		public static RoutedCommand azzi920Command
		{
			get
			{
				if (MenuCommands._azzi920Command == null)
				{
					MenuCommands._azzi920Command = new RoutedCommand("azzi920Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi920Command;
			}
		}

		// Token: 0x0600011E RID: 286 RVA: 0x000058B8 File Offset: 0x00003AB8
		public static void CanExecuteazzi920(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600011F RID: 287 RVA: 0x000058C1 File Offset: 0x00003AC1
		public static void Executedazzi920(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("azzi920");
		}

		// Token: 0x1700004E RID: 78
		// (get) Token: 0x06000120 RID: 288 RVA: 0x000058CE File Offset: 0x00003ACE
		public static RoutedCommand adzp168Command
		{
			get
			{
				if (MenuCommands._adzp168Command == null)
				{
					MenuCommands._adzp168Command = new RoutedCommand("adzp168Command", typeof(MenuCommands));
				}
				return MenuCommands._adzp168Command;
			}
		}

		// Token: 0x06000121 RID: 289 RVA: 0x000058F5 File Offset: 0x00003AF5
		public static void CanExecuteadzp168(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000122 RID: 290 RVA: 0x000058FE File Offset: 0x00003AFE
		public static void Executedadzp168(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzp168");
		}

		// Token: 0x1700004F RID: 79
		// (get) Token: 0x06000123 RID: 291 RVA: 0x0000590B File Offset: 0x00003B0B
		public static RoutedCommand adzp165Command
		{
			get
			{
				if (MenuCommands._adzp165Command == null)
				{
					MenuCommands._adzp165Command = new RoutedCommand("adzp165Command", typeof(MenuCommands));
				}
				return MenuCommands._adzp165Command;
			}
		}

		// Token: 0x06000124 RID: 292 RVA: 0x00005932 File Offset: 0x00003B32
		public static void CanExecuteadzp165(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000125 RID: 293 RVA: 0x00005956 File Offset: 0x00003B56
		public static void Executedadzp165(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzp165");
		}

		// Token: 0x17000050 RID: 80
		// (get) Token: 0x06000126 RID: 294 RVA: 0x00005963 File Offset: 0x00003B63
		public static RoutedCommand adzq255Command
		{
			get
			{
				if (MenuCommands._adzq255Command == null)
				{
					MenuCommands._adzq255Command = new RoutedCommand("adzq255Command", typeof(MenuCommands));
				}
				return MenuCommands._adzq255Command;
			}
		}

		// Token: 0x06000127 RID: 295 RVA: 0x0000598A File Offset: 0x00003B8A
		public static void CanExecuteadzq255(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000128 RID: 296 RVA: 0x00005993 File Offset: 0x00003B93
		public static void Executedadzq255(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzq255");
		}

		// Token: 0x17000051 RID: 81
		// (get) Token: 0x06000129 RID: 297 RVA: 0x000059A0 File Offset: 0x00003BA0
		public static RoutedCommand ReportDownloadCommand
		{
			get
			{
				if (MenuCommands._reportDownloadCommand == null)
				{
					MenuCommands._reportDownloadCommand = new RoutedCommand("ReportDownloadCommand", typeof(MenuCommands));
				}
				return MenuCommands._reportDownloadCommand;
			}
		}

		// Token: 0x0600012A RID: 298 RVA: 0x000059C7 File Offset: 0x00003BC7
		public static void CanExecuteReportDownload(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600012B RID: 299 RVA: 0x000059D0 File Offset: 0x00003BD0
		public static void ExecutedReportDownload(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.ReportTemplateDownload(SettingManager.Get().CurrentSetting.Connection.Workspace);
		}

		// Token: 0x17000052 RID: 82
		// (get) Token: 0x0600012C RID: 300 RVA: 0x000059EC File Offset: 0x00003BEC
		public static RoutedCommand ReportUploadCommand
		{
			get
			{
				if (MenuCommands._reportUploadCommand == null)
				{
					MenuCommands._reportUploadCommand = new RoutedCommand("ReportUploadCommand", typeof(MenuCommands));
				}
				return MenuCommands._reportUploadCommand;
			}
		}

		// Token: 0x0600012D RID: 301 RVA: 0x00005A13 File Offset: 0x00003C13
		public static void CanExecuteReportUpload(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600012E RID: 302 RVA: 0x00005A1C File Offset: 0x00003C1C
		public static void ExecutedReportUpload(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.UploadReportTemplate();
		}

		// Token: 0x17000053 RID: 83
		// (get) Token: 0x0600012F RID: 303 RVA: 0x00005A24 File Offset: 0x00003C24
		public static RoutedCommand ReportOpenCommand
		{
			get
			{
				if (MenuCommands._reportOpenCommand == null)
				{
					MenuCommands._reportOpenCommand = new RoutedCommand("ReportOpenCommand", typeof(MenuCommands));
				}
				return MenuCommands._reportOpenCommand;
			}
		}

		// Token: 0x06000130 RID: 304 RVA: 0x00005A4B File Offset: 0x00003C4B
		public static void CanExecuteReportOpen(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000131 RID: 305 RVA: 0x00005A54 File Offset: 0x00003C54
		public static void ExecutedReportOpen(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenReportTemplate();
		}

		// Token: 0x17000054 RID: 84
		// (get) Token: 0x06000132 RID: 306 RVA: 0x00005A67 File Offset: 0x00003C67
		public static RoutedCommand adzp270Command
		{
			get
			{
				if (MenuCommands._adzp270Command == null)
				{
					MenuCommands._adzp270Command = new RoutedCommand("adzp270Command", typeof(MenuCommands));
				}
				return MenuCommands._adzp270Command;
			}
		}

		// Token: 0x06000133 RID: 307 RVA: 0x00005A8E File Offset: 0x00003C8E
		public static void CanExecuteadzp270(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000134 RID: 308 RVA: 0x00005A97 File Offset: 0x00003C97
		public static void Executedadzp270(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.RunProgram("adzp270");
		}

		// Token: 0x17000055 RID: 85
		// (get) Token: 0x06000135 RID: 309 RVA: 0x00005AAB File Offset: 0x00003CAB
		public static RoutedCommand PreferenceWindowCommand
		{
			get
			{
				if (MenuCommands._preferenceWindowCommand == null)
				{
					MenuCommands._preferenceWindowCommand = new RoutedCommand("PreferenceWindowCommand", typeof(MenuCommands));
				}
				return MenuCommands._preferenceWindowCommand;
			}
		}

		// Token: 0x06000136 RID: 310 RVA: 0x00005AD2 File Offset: 0x00003CD2
		public static void CanExecutePreferenceWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000137 RID: 311 RVA: 0x00005ADC File Offset: 0x00003CDC
		public static void ExecutedPreferenceWindow(object sender, ExecutedRoutedEventArgs e)
		{
			new PreferenceWindow
			{
				Owner = Application.Current.MainWindow
			}.ShowDialog();
		}

		// Token: 0x17000056 RID: 86
		// (get) Token: 0x06000138 RID: 312 RVA: 0x00005B06 File Offset: 0x00003D06
		public static RoutedCommand RecentFilesCommand
		{
			get
			{
				if (MenuCommands._recentFilesCommand == null)
				{
					MenuCommands._recentFilesCommand = new RoutedCommand("RecentFilesCommand", typeof(MenuCommands));
				}
				return MenuCommands._recentFilesCommand;
			}
		}

		// Token: 0x06000139 RID: 313 RVA: 0x00005B2D File Offset: 0x00003D2D
		public static void CanExecuteRecentFiles(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600013A RID: 314 RVA: 0x00005B38 File Offset: 0x00003D38
		public static void ExecutedRecentFiles(object sender, ExecutedRoutedEventArgs e)
		{
			string text = e.Parameter as string;
			if (!string.IsNullOrEmpty(text))
			{
				if (text.StartsWith("_"))
				{
					text = text.Remove(0, 1);
				}
				SettingManager.Get().OpenSpecFiles(text);
			}
		}

		// Token: 0x17000057 RID: 87
		// (get) Token: 0x0600013B RID: 315 RVA: 0x00005B7A File Offset: 0x00003D7A
		public static RoutedCommand adzp990ExpCommand
		{
			get
			{
				if (MenuCommands._adzp990ExpCommand == null)
				{
					MenuCommands._adzp990ExpCommand = new RoutedCommand("adzp990ExpCommand", typeof(MenuCommands));
				}
				return MenuCommands._adzp990ExpCommand;
			}
		}

		// Token: 0x0600013C RID: 316 RVA: 0x00005BA1 File Offset: 0x00003DA1
		public static void CanExecuteadzp990Exp(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600013D RID: 317 RVA: 0x00005BAA File Offset: 0x00003DAA
		public static void Executedadzp990Exp(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzp990 exp");
		}

		// Token: 0x17000058 RID: 88
		// (get) Token: 0x0600013E RID: 318 RVA: 0x00005BB7 File Offset: 0x00003DB7
		public static RoutedCommand adzp990ImpCommand
		{
			get
			{
				if (MenuCommands._adzp990ImpCommand == null)
				{
					MenuCommands._adzp990ImpCommand = new RoutedCommand("adzp990ImpCommand", typeof(MenuCommands));
				}
				return MenuCommands._adzp990ImpCommand;
			}
		}

		// Token: 0x0600013F RID: 319 RVA: 0x00005BDE File Offset: 0x00003DDE
		public static void CanExecuteadzp990Imp(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000140 RID: 320 RVA: 0x00005BE7 File Offset: 0x00003DE7
		public static void Executedadzp990Imp(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzp990 imp");
		}

		// Token: 0x17000059 RID: 89
		// (get) Token: 0x06000141 RID: 321 RVA: 0x00005BF4 File Offset: 0x00003DF4
		public static RoutedCommand adzi888ExpCommand
		{
			get
			{
				if (MenuCommands._adzi888ExpCommand == null)
				{
					MenuCommands._adzi888ExpCommand = new RoutedCommand("adzi888ExpCommand", typeof(MenuCommands));
				}
				return MenuCommands._adzi888ExpCommand;
			}
		}

		// Token: 0x06000142 RID: 322 RVA: 0x00005C1B File Offset: 0x00003E1B
		public static void CanExecuteadzi888Exp(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000143 RID: 323 RVA: 0x00005C24 File Offset: 0x00003E24
		public static void Executedadzi888Exp(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi888 simple_all");
		}

		// Token: 0x1700005A RID: 90
		// (get) Token: 0x06000144 RID: 324 RVA: 0x00005C31 File Offset: 0x00003E31
		public static RoutedCommand adzi888ImpCommand
		{
			get
			{
				if (MenuCommands._adzi888ImpCommand == null)
				{
					MenuCommands._adzi888ImpCommand = new RoutedCommand("adzi888ImpCommand", typeof(MenuCommands));
				}
				return MenuCommands._adzi888ImpCommand;
			}
		}

		// Token: 0x06000145 RID: 325 RVA: 0x00005C58 File Offset: 0x00003E58
		public static void CanExecuteadzi888Imp(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000146 RID: 326 RVA: 0x00005C61 File Offset: 0x00003E61
		public static void Executedadzi888Imp(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzi888 simple_all quick_imp");
		}

		// Token: 0x1700005B RID: 91
		// (get) Token: 0x06000147 RID: 327 RVA: 0x00005C6E File Offset: 0x00003E6E
		public static RoutedCommand azzi909Command
		{
			get
			{
				if (MenuCommands._azzi909Command == null)
				{
					MenuCommands._azzi909Command = new RoutedCommand("azzi909Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi909Command;
			}
		}

		// Token: 0x06000148 RID: 328 RVA: 0x00005C98 File Offset: 0x00003E98
		public static void CanExecuteazzi909(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && (MenuCommands.ProgramKey.PackType == TzpType.Code || MenuCommands.ProgramKey.PackType == TzpType.ReportCode))
			{
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey);
				if (tzpManger.Booking)
				{
					ProgramInformation programInfo = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey);
					if (programInfo != null && programInfo.GetCodeTemplate().Equals("Q", StringComparison.CurrentCultureIgnoreCase))
					{
						flag = true;
					}
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000149 RID: 329 RVA: 0x00005D18 File Offset: 0x00003F18
		public static void Executedazzi909(object sender, ExecutedRoutedEventArgs e)
		{
			if (MessageBoxResult.Yes == DesignerMessageBox.Show(Application.Current.FindResource("Message_QSetting") as string, string.Format("{0} [{1}]", Application.Current.FindResource("menu_QSetting") as string, MenuCommands.ProgramKey.Program), MessageBoxButton.YesNo, MessageBoxImage.Exclamation))
			{
				ConnectionManager.RunProgram(string.Format("azzi909 {0} tiptop", MenuCommands.ProgramKey.Program));
			}
		}

		// Token: 0x1700005C RID: 92
		// (get) Token: 0x0600014A RID: 330 RVA: 0x00005D86 File Offset: 0x00003F86
		public static RoutedCommand FormPreviewCommand
		{
			get
			{
				if (MenuCommands._formPreviewCommand == null)
				{
					MenuCommands._formPreviewCommand = new RoutedCommand("FormPreviewCommand", typeof(MenuCommands));
				}
				return MenuCommands._formPreviewCommand;
			}
		}

		// Token: 0x0600014B RID: 331 RVA: 0x00005DB0 File Offset: 0x00003FB0
		public static void CanExecuteFormPreview(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (MenuCommands.ProgramKey != null && MenuCommands.ProgramKey.PackType == TzpType.Form)
			{
				flag = true;
			}
			e.CanExecute = flag;
		}

		// Token: 0x0600014C RID: 332 RVA: 0x00005DE4 File Offset: 0x00003FE4
		public static void ExecutedFormPreview(object sender, ExecutedRoutedEventArgs e)
		{
			if (MessageBoxResult.OK == DesignerMessageBox.Show(Application.Current.FindResource("Message_PreviewForm") as string, Application.Current.FindResource("menu_PreviewForm") as string, MessageBoxButton.OK, MessageBoxImage.Exclamation))
			{
				ConnectionManager.PreviewForm(MenuCommands.ProgramKey.Program);
			}
		}

		// Token: 0x1700005D RID: 93
		// (get) Token: 0x0600014D RID: 333 RVA: 0x00005E34 File Offset: 0x00004034
		public static RoutedCommand ShowGeneroUserGuideCommand
		{
			get
			{
				if (MenuCommands._showGeneroUserGuideCommand == null)
				{
					MenuCommands._showGeneroUserGuideCommand = new RoutedCommand("ShowGeneroUserGuideCommand", typeof(MenuCommands));
				}
				return MenuCommands._showGeneroUserGuideCommand;
			}
		}

		// Token: 0x0600014E RID: 334 RVA: 0x00005E5B File Offset: 0x0000405B
		public static void CanExecuteShowGeneroUserGuide(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600014F RID: 335 RVA: 0x00005E64 File Offset: 0x00004064
		public static void ExecutedShowGeneroUserGuide(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			Process.Start(SettingManager.Get().SiteManager.SelectedSetting.Connection.UserGuideURL);
		}

		// Token: 0x1700005E RID: 94
		// (get) Token: 0x06000150 RID: 336 RVA: 0x00005E8C File Offset: 0x0000408C
		public static RoutedCommand ShowTOPSTDLoginCommand
		{
			get
			{
				if (MenuCommands._showTOPSTDLoginCommand == null)
				{
					MenuCommands._showTOPSTDLoginCommand = new RoutedCommand("ShowTOPSTDLoginCommand", typeof(MenuCommands));
				}
				return MenuCommands._showTOPSTDLoginCommand;
			}
		}

		// Token: 0x06000151 RID: 337 RVA: 0x00005EB3 File Offset: 0x000040B3
		public static void CanExecuteShowTOPSTDLogin(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000152 RID: 338 RVA: 0x00005EBC File Offset: 0x000040BC
		public static void ExecutedShowTOPSTDLogin(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TOPSTDSettingModel topstdSetting = SettingManager.Get().TopstdSetting;
			if (new TOPSTDLoginWindow(topstdSetting).ShowDialog() == true)
			{
				topstdSetting.IsLogin = ConnectionManager.VerifyAccount(topstdSetting.Login, topstdSetting.Password);
			}
		}

		// Token: 0x1700005F RID: 95
		// (get) Token: 0x06000153 RID: 339 RVA: 0x00005F15 File Offset: 0x00004115
		public static RoutedCommand SpecCheckInCommand
		{
			get
			{
				if (MenuCommands._specCheckInCommand == null)
				{
					MenuCommands._specCheckInCommand = new RoutedCommand("SpecCheckInCommand", typeof(MenuCommands));
				}
				return MenuCommands._specCheckInCommand;
			}
		}

		// Token: 0x06000154 RID: 340 RVA: 0x00005F3C File Offset: 0x0000413C
		public static void CanExecuteSpecCheckIn(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null)
			{
				if (!SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type.Equals(TzpType.Form))
				{
					flag = false;
				}
				else if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "M" || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "S" || SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).ProgType == "Q")
				{
					if (e.Parameter != null && e.Parameter is TzpType)
					{
						TzpType? tzpType = new TzpType?((TzpType)e.Parameter);
						flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == tzpType && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
					}
					else
					{
						flag = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking;
					}
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000155 RID: 341 RVA: 0x000060A4 File Offset: 0x000042A4
		public static void ExecutedSpecCheckIn(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.SpecCheckIn(MenuCommands.ProgramKey);
		}

		// Token: 0x17000060 RID: 96
		// (get) Token: 0x06000156 RID: 342 RVA: 0x000060B0 File Offset: 0x000042B0
		public static RoutedCommand adzq001Command
		{
			get
			{
				if (MenuCommands._adzq001Command == null)
				{
					MenuCommands._adzq001Command = new RoutedCommand("adzq001Command", typeof(MenuCommands));
				}
				return MenuCommands._adzq001Command;
			}
		}

		// Token: 0x06000157 RID: 343 RVA: 0x000060D7 File Offset: 0x000042D7
		public static void CanExecuteadzq001(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000158 RID: 344 RVA: 0x000060E0 File Offset: 0x000042E0
		public static void Executedadzq001(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzq001");
		}

		// Token: 0x17000061 RID: 97
		// (get) Token: 0x06000159 RID: 345 RVA: 0x000060ED File Offset: 0x000042ED
		public static RoutedCommand adzq003Command
		{
			get
			{
				if (MenuCommands._adzq003Command == null)
				{
					MenuCommands._adzq003Command = new RoutedCommand("adzq003Command", typeof(MenuCommands));
				}
				return MenuCommands._adzq003Command;
			}
		}

		// Token: 0x0600015A RID: 346 RVA: 0x00006114 File Offset: 0x00004314
		public static void CanExecuteadzq003(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600015B RID: 347 RVA: 0x0000611D File Offset: 0x0000431D
		public static void Executedadzq003(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzq003");
		}

		// Token: 0x17000062 RID: 98
		// (get) Token: 0x0600015C RID: 348 RVA: 0x0000612C File Offset: 0x0000432C
		public static RoutedCommand FileOpenSpecCommand
		{
			get
			{
				if (MenuCommands._fileOpenSpecCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.O, ModifierKeys.Control));
					MenuCommands._fileOpenSpecCommand = new RoutedCommand("FileOpenSpecCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._fileOpenSpecCommand;
			}
		}

		// Token: 0x0600015D RID: 349 RVA: 0x00006174 File Offset: 0x00004374
		public static void CanExecuteFileOpenSpec(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x0600015E RID: 350 RVA: 0x0000618D File Offset: 0x0000438D
		public static void ExecutedFileOpenSpec(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenFile(1);
		}

		// Token: 0x17000063 RID: 99
		// (get) Token: 0x0600015F RID: 351 RVA: 0x000061A4 File Offset: 0x000043A4
		public static RoutedCommand FileOpenCodeCommand
		{
			get
			{
				if (MenuCommands._fileOpenCodeCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.O, ModifierKeys.Control));
					MenuCommands._fileOpenCodeCommand = new RoutedCommand("FileOpenCodeCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._fileOpenCodeCommand;
			}
		}

		// Token: 0x06000160 RID: 352 RVA: 0x000061EC File Offset: 0x000043EC
		public static void CanExecuteFileOpenCode(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().CommonDataIsLoaded;
		}

		// Token: 0x06000161 RID: 353 RVA: 0x00006205 File Offset: 0x00004405
		public static void ExecutedFileOpenCode(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenFile(2);
		}

		// Token: 0x17000064 RID: 100
		// (get) Token: 0x06000162 RID: 354 RVA: 0x00006219 File Offset: 0x00004419
		public static RoutedCommand ShowServiceCloudLoginCommand
		{
			get
			{
				if (MenuCommands._showServiceCloudLoginCommand == null)
				{
					MenuCommands._showServiceCloudLoginCommand = new RoutedCommand("ShowServiceCloudLoginCommand", typeof(MenuCommands));
				}
				return MenuCommands._showServiceCloudLoginCommand;
			}
		}

		// Token: 0x06000163 RID: 355 RVA: 0x00006240 File Offset: 0x00004440
		public static void CanExecuteShowServiceCloudLogin(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
			if (ConnectionInfo.This.LoginUser != "topstd" && ConnectionInfo.This.LoginUser != "topman" && ConnectionInfo.This.LoginUser != "topapp")
			{
				e.CanExecute = false;
			}
		}

		// Token: 0x06000164 RID: 356 RVA: 0x000062A0 File Offset: 0x000044A0
		public static void ExecutedShowServiceCloudLogin(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ServiceCloudLoginWindow serviceCloudLoginWindow = new ServiceCloudLoginWindow();
			serviceCloudLoginWindow.ShowDialog();
		}

		// Token: 0x17000065 RID: 101
		// (get) Token: 0x06000165 RID: 357 RVA: 0x000062C1 File Offset: 0x000044C1
		public static RoutedCommand ToggleLayoutCommand
		{
			get
			{
				if (MenuCommands._toggleLayoutCommand == null)
				{
					MenuCommands._toggleLayoutCommand = new RoutedCommand("ToggleLayoutCommand", typeof(MenuCommands));
				}
				return MenuCommands._toggleLayoutCommand;
			}
		}

		// Token: 0x06000166 RID: 358 RVA: 0x000062E8 File Offset: 0x000044E8
		public static void CanExecuteToggleLayout(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000167 RID: 359 RVA: 0x000062F4 File Offset: 0x000044F4
		public static void ExecutedToggleLayout(object sender, ExecutedRoutedEventArgs e)
		{
			SpecDesignerMainWindow specDesignerMainWindow = sender as SpecDesignerMainWindow;
			if (specDesignerMainWindow != null)
			{
				string text = e.Parameter as string;
				specDesignerMainWindow.ShowSelectedLayout(text);
			}
		}

		// Token: 0x17000066 RID: 102
		// (get) Token: 0x06000168 RID: 360 RVA: 0x0000631E File Offset: 0x0000451E
		public static RoutedCommand ToggleLiteCommand
		{
			get
			{
				if (MenuCommands._toggleLiteCommand == null)
				{
					MenuCommands._toggleLiteCommand = new RoutedCommand("ToggleLiteCommand", typeof(MenuCommands));
				}
				return MenuCommands._toggleLiteCommand;
			}
		}

		// Token: 0x06000169 RID: 361 RVA: 0x00006345 File Offset: 0x00004545
		public static void CanExecuteToggleLite(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x0600016A RID: 362 RVA: 0x00006350 File Offset: 0x00004550
		public static void ExecutedToggleLite(object sender, ExecutedRoutedEventArgs e)
		{
			SpecDesignerMainWindow specDesignerMainWindow = sender as SpecDesignerMainWindow;
			if (specDesignerMainWindow != null)
			{
				specDesignerMainWindow.ShowLiteLayout();
			}
		}

		// Token: 0x17000067 RID: 103
		// (get) Token: 0x0600016B RID: 363 RVA: 0x0000636D File Offset: 0x0000456D
		public static RoutedCommand ViewFormDiffListCommand
		{
			get
			{
				if (MenuCommands._viewFormDiffListCommand == null)
				{
					MenuCommands._viewFormDiffListCommand = new RoutedCommand("ViewFormDiffListCommand", typeof(MenuCommands));
				}
				return MenuCommands._viewFormDiffListCommand;
			}
		}

		// Token: 0x0600016C RID: 364 RVA: 0x00006394 File Offset: 0x00004594
		public static void CanExecuteViewFormDiffList(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).FormDiffList != null;
		}

		// Token: 0x0600016D RID: 365 RVA: 0x000063E9 File Offset: 0x000045E9
		public static void ExecutedViewFormDiffList(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			DiffListWindow.This.DataContext = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).FormDiffList;
			DiffListWindow.This.Show();
		}

		// Token: 0x17000068 RID: 104
		// (get) Token: 0x0600016E RID: 366 RVA: 0x0000641C File Offset: 0x0000461C
		public static RoutedCommand NextDiffCommand
		{
			get
			{
				if (MenuCommands._nextdiffCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.O, ModifierKeys.Control));
					MenuCommands._nextdiffCommand = new RoutedCommand("NextDiffCommand", typeof(MenuCommands), inputGestureCollection);
				}
				return MenuCommands._nextdiffCommand;
			}
		}

		// Token: 0x0600016F RID: 367 RVA: 0x00006464 File Offset: 0x00004664
		public static void CanExecuteNextDiff(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).IsDiff;
		}

		// Token: 0x06000170 RID: 368 RVA: 0x00006487 File Offset: 0x00004687
		public static void ExecutedNextDiff(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
		}

		// Token: 0x17000069 RID: 105
		// (get) Token: 0x06000171 RID: 369 RVA: 0x00006490 File Offset: 0x00004690
		public static RoutedCommand SearchKeySettingWindowCommand
		{
			get
			{
				if (MenuCommands._searchKeySettingWindowCommand == null)
				{
					MenuCommands._searchKeySettingWindowCommand = new RoutedCommand("SearchKeySettingWindowCommand", typeof(MenuCommands));
				}
				return MenuCommands._searchKeySettingWindowCommand;
			}
		}

		// Token: 0x06000172 RID: 370 RVA: 0x000064B7 File Offset: 0x000046B7
		public static void CanExecuteSearchKeySettingWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			if (!(MenuCommands.ProgramKey != null))
			{
				e.CanExecute = false;
				return;
			}
			if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).IsDiff)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x06000173 RID: 371 RVA: 0x000064F4 File Offset: 0x000046F4
		public static void ExecutedSearchKeySettingWindow(object sender, ExecutedRoutedEventArgs e)
		{
			new SearchKeySettingWindow(MenuCommands.ProgramKey)
			{
				Owner = Application.Current.MainWindow
			}.ShowDialog();
		}

		// Token: 0x1700006A RID: 106
		// (get) Token: 0x06000174 RID: 372 RVA: 0x00006524 File Offset: 0x00004724
		public static PackageKey ProgramKey
		{
			get
			{
				return (EditorWorkspace.This.ActiveDocument != null) ? EditorWorkspace.This.ActiveDocument.Key : null;
			}
		}

		// Token: 0x1700006B RID: 107
		// (get) Token: 0x06000175 RID: 373 RVA: 0x00006554 File Offset: 0x00004754
		public static string ZipFile
		{
			get
			{
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey);
				if (tzpManger == null)
				{
					return string.Empty;
				}
				return tzpManger.ZipFile;
			}
		}

		// Token: 0x1700006C RID: 108
		// (get) Token: 0x06000176 RID: 374 RVA: 0x00006580 File Offset: 0x00004780
		public static string ModuleName
		{
			get
			{
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey);
				if (tzpManger == null)
				{
					return string.Empty;
				}
				return tzpManger.ModuleName;
			}
		}

		// Token: 0x06000177 RID: 375 RVA: 0x000065AC File Offset: 0x000047AC
		private static bool Confirm(TzpType type, bool isOnlyUpload)
		{
			string text = string.Empty;
			string text2 = string.Empty;
			switch (type)
			{
			case TzpType.ReportSpec:
			case TzpType.CodeSpec:
			case TzpType.Form:
			{
				text = (isOnlyUpload ? (Application.Current.FindResource("Message_ConfirmBeforeUpload_SD") as string) : (Application.Current.FindResource("Message_ConfirmBeforePrecompile_SD") as string));
				text = string.Format(text, MenuCommands.ProgramKey.Program);
				text2 = ((type == TzpType.Form) ? SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).SpecificationInfo.TSDElement.Attribute("type").Value : string.Empty);
				string text3;
				if ((text3 = text2.ToUpper()) != null)
				{
					if (!(text3 == "S"))
					{
						if (!(text3 == "F") && !(text3 == "Q"))
						{
							if (text3 == "M")
							{
								text = string.Format("{0}\n\n{1}\n{2}", text, Application.Current.FindResource("Message_UploadWarning") as string, Application.Current.FindResource("Message_FormUploaWarning2") as string);
							}
						}
						else
						{
							text = string.Format("{0}\n\n{1}\n{2}", text, Application.Current.FindResource("Message_UploadWarning") as string, Application.Current.FindResource("Message_FormUploaWarning") as string);
						}
					}
					else
					{
						text = string.Format("{0}\n\n{1}\n{2}\n\n<< {3} >>", new object[]
						{
							text,
							Application.Current.FindResource("Message_UploadWarning") as string,
							Application.Current.FindResource("Message_FormUploaWarning") as string,
							Application.Current.FindResource("Message_FormUploaWarning2") as string
						});
					}
				}
				break;
			}
			case TzpType.ReportCode:
			case TzpType.Code:
			{
				text = (isOnlyUpload ? (Application.Current.FindResource("Message_ConfirmBeforeUpload_PR") as string) : (Application.Current.FindResource("Message_ConfirmBeforePrecompile_PR") as string));
				text = string.Format(text, MenuCommands.ProgramKey.Program);
				string text4 = string.Empty;
				text2 = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).TAP.Attribute("type").Value;
				if (ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).IsSectionModify)
				{
					text4 = Application.Current.FindResource("Message_SectionBreak") as string;
				}
				if ((text2.Equals("M", StringComparison.CurrentCultureIgnoreCase) || text2.Equals("S", StringComparison.CurrentCultureIgnoreCase)) && !SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).IsNormalStyle)
				{
					text4 = ((!string.IsNullOrEmpty(text4)) ? string.Format("{0} / FreeStyle ", text4) : (Application.Current.FindResource("Message_FreeStyle") as string));
				}
				else if (text2.Equals("B", StringComparison.CurrentCultureIgnoreCase) || text2.Equals("W", StringComparison.CurrentCultureIgnoreCase) || text2.Equals("Q", StringComparison.CurrentCultureIgnoreCase))
				{
					text4 = string.Format(Application.Current.FindResource("Message_CodeUploaWarning") as string, MenuCommands.ProgramKey.Program);
				}
				if (text4.Length > 0)
				{
					text = string.Format("{0}\n\n{1}\n{2}", text, Application.Current.FindResource("Message_UploadWarning") as string, text4);
				}
				break;
			}
			default:
				text = "確認上傳？";
				break;
			}
			MessageBoxResult messageBoxResult = DesignerMessageBox.Show(text, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question);
			if (messageBoxResult != MessageBoxResult.Yes)
			{
				return false;
			}
			if (EditorWorkspace.This.ActiveDocument.IsModified)
			{
				MessageBoxResult messageBoxResult2 = ((!PreferenceManager.Current.Settings.RemaindSave) ? MessageBoxResult.Yes : DesignerMessageBox.Show(Application.Current.FindResource("Message_SaveBeforeUpload") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question));
				if (MessageBoxResult.Yes == messageBoxResult2)
				{
					if (isOnlyUpload)
					{
						SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).TzpSaved += MenuCommands.OnTzpSavedBeforeUpload;
					}
					else
					{
						SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).TzpSaved += MenuCommands.OnTzpSavedBeforePrecompile;
					}
					SettingManager.Get().SaveSetting(MenuCommands.ProgramKey);
					return false;
				}
			}
			return true;
		}

		// Token: 0x06000178 RID: 376 RVA: 0x000069DA File Offset: 0x00004BDA
		private static void ConfirmSaveAndUpload(TzpType type, bool CodeBooking)
		{
			if (!MenuCommands.Confirm(type, true))
			{
				return;
			}
			MenuCommands.UploadBasedOnType(type, true);
		}

		// Token: 0x06000179 RID: 377 RVA: 0x000069ED File Offset: 0x00004BED
		private static void ConfirmSaveAndUpload(TzpType type)
		{
			if (!MenuCommands.Confirm(type, true))
			{
				return;
			}
			MenuCommands.UploadBasedOnType(type);
		}

		// Token: 0x0600017A RID: 378 RVA: 0x000069FF File Offset: 0x00004BFF
		private static void ConfirmSaveAndPreCompile(TzpType type)
		{
			if (!MenuCommands.Confirm(type, false))
			{
				return;
			}
			MenuCommands.PrecompileBaseOnType(type);
		}

		// Token: 0x0600017B RID: 379 RVA: 0x00006A11 File Offset: 0x00004C11
		private static void OnTzpSavedBeforeUpload(object sender, TzpSavedEventArgs e)
		{
			SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).TzpSaved -= MenuCommands.OnTzpSavedBeforeUpload;
			MenuCommands.UploadBasedOnType(e.Type);
		}

		// Token: 0x0600017C RID: 380 RVA: 0x00006A3E File Offset: 0x00004C3E
		private static void OnTzpSavedBeforePrecompile(object sender, TzpSavedEventArgs e)
		{
			SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).TzpSaved -= MenuCommands.OnTzpSavedBeforePrecompile;
			MenuCommands.PrecompileBaseOnType(e.Type);
		}

		// Token: 0x0600017D RID: 381 RVA: 0x00006A6C File Offset: 0x00004C6C
		private static void PrecompileBaseOnType(TzpType type)
		{
			switch (type)
			{
			case TzpType.Code:
				ConnectionManager.UploadCode(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "C4GL");
				return;
			case TzpType.CodeSpec:
				break;
			case TzpType.Form:
				ConnectionManager.UploadSpec(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "C4FD");
				break;
			default:
				return;
			}
		}

		// Token: 0x0600017E RID: 382 RVA: 0x00006AD0 File Offset: 0x00004CD0
		private static void UploadBasedOnType(TzpType type, bool CodeBooking)
		{
			if (type != TzpType.Form)
			{
				return;
			}
			MenuCommands.UpdateSpecification(true);
		}

		// Token: 0x0600017F RID: 383 RVA: 0x00006AEC File Offset: 0x00004CEC
		private static void UploadBasedOnType(TzpType type)
		{
			switch (type)
			{
			case TzpType.ReportSpec:
				MenuCommands.UpdateReportSpecification();
				return;
			case TzpType.ReportCode:
			case TzpType.Code:
				MenuCommands.UpdateProgram(type);
				return;
			case TzpType.CodeSpec:
				MenuCommands.UpdateCodeSpecification();
				return;
			case TzpType.Form:
				MenuCommands.UpdateSpecification(false);
				return;
			default:
				return;
			}
		}

		// Token: 0x06000180 RID: 384 RVA: 0x00006B34 File Offset: 0x00004D34
		private static void UpdateProgram(TzpType type)
		{
			string value = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).TAP.Attribute("type").Value;
			if (value.Equals("G", StringComparison.CurrentCultureIgnoreCase))
			{
				ConnectionManager.UploadCode(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, TzpType.ReportCode.Description());
				return;
			}
			ConnectionManager.UploadCode(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, TzpType.Code.Description());
		}

		// Token: 0x06000181 RID: 385 RVA: 0x00006BC3 File Offset: 0x00004DC3
		private static void UpdateCodeSpecification()
		{
			ConnectionManager.UploadSpec(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "CSPEC");
		}

		// Token: 0x06000182 RID: 386 RVA: 0x00006BE4 File Offset: 0x00004DE4
		private static void UpdateReportSpecification()
		{
			ConnectionManager.UploadSpec(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "RSPEC");
		}

		// Token: 0x06000183 RID: 387 RVA: 0x00006C08 File Offset: 0x00004E08
		private static void UpdateSpecification(bool CodeBooking)
		{
			if (CodeBooking)
			{
				ConnectionManager.UploadSpec(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "SPECRC3");
				return;
			}
			ConnectionManager.UploadSpec(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "SPEC");
		}

		// Token: 0x1700006D RID: 109
		// (get) Token: 0x06000184 RID: 388 RVA: 0x00006C57 File Offset: 0x00004E57
		public static RoutedCommand SRRelationSettingCommand
		{
			get
			{
				if (MenuCommands._srRelationSettingCommand == null)
				{
					MenuCommands._srRelationSettingCommand = new RoutedCommand("SRRelationSettingCommand", typeof(MenuCommands));
				}
				return MenuCommands._srRelationSettingCommand;
			}
		}

		// Token: 0x06000185 RID: 389 RVA: 0x00006C80 File Offset: 0x00004E80
		public static void ExecutedSRRelationSetting(object sender, ExecutedRoutedEventArgs e)
		{
			Window window = new Window();
			window.Title = string.Format("{0} [{1}]", Application.Current.FindResource("menu_TableSchemaAssocSetting") as string, MenuCommands.ProgramKey.Program);
			window.Topmost = true;
			window.Width = 400.0;
			window.Height = 500.0;
			window.Content = new RelationshipSpace();
			window.Show();
			window.Closed += MenuCommands.win_Closed;
		}

		// Token: 0x06000186 RID: 390 RVA: 0x00006D0C File Offset: 0x00004F0C
		private static void win_Closed(object sender, EventArgs e)
		{
			Window window = sender as Window;
			window.Closed -= MenuCommands.win_Closed;
			RelationshipSpace relationshipSpace = window.DataContext as RelationshipSpace;
			if (relationshipSpace != null)
			{
				relationshipSpace.Dispose();
			}
			window.DataContext = null;
		}

		// Token: 0x06000187 RID: 391 RVA: 0x00006D50 File Offset: 0x00004F50
		public static void CanExecuteSRRelationSetting(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (MenuCommands.ProgramKey != null)
			{
				e.CanExecute = SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type == TzpType.Form;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x1700006E RID: 110
		// (get) Token: 0x06000188 RID: 392 RVA: 0x00006DAA File Offset: 0x00004FAA
		public static RoutedCommand SetFreeStyleCommand
		{
			get
			{
				if (MenuCommands._setFreeStyleCommand == null)
				{
					MenuCommands._setFreeStyleCommand = new RoutedCommand("SetFreeStyleCommand", typeof(MenuCommands));
				}
				return MenuCommands._setFreeStyleCommand;
			}
		}

		// Token: 0x06000189 RID: 393 RVA: 0x00006DD4 File Offset: 0x00004FD4
		public static void ExecutedSetFreeStyle(object sender, ExecutedRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			if (@this.ActiveDocument != null)
			{
				if (ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).login_user == "topstd")
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_TopstdCantSetFreestyle") as string);
					return;
				}
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				string text = string.Format("{0}{1}", Application.Current.FindResource("Message_SetFreeStyleWarning") as string, Environment.NewLine);
				if (DesignerMessageBox.Show(text, string.Format("Free Style [{0}]", activeDocument.Key.Program), MessageBoxButton.YesNo, MessageBoxImage.Exclamation) == MessageBoxResult.Yes)
				{
					switch (tzpManger.Type)
					{
					case TzpType.ReportCode:
					case TzpType.Code:
					{
						string text2 = string.Format("{0}\\", Path.GetDirectoryName(MenuCommands.ZipFile)).Replace("\\", "\\\\");
						ConnectionManager.RunProgram(string.Format("adzp085 '{0}' {1} {2} '{3}' '{4}' '{5}' '{6}'", new object[]
						{
							Path.GetFileName(MenuCommands.ZipFile),
							MenuCommands.ModuleName,
							"CODE",
							text2,
							"",
							SettingManager.Get().GetTzpManger(activeDocument.Key).ProgIdentity,
							SettingManager.Get().GetTzpManger(activeDocument.Key).ProgVer
						}));
						return;
					}
					case TzpType.CodeSpec:
						break;
					case TzpType.Form:
						tzpManger.SpecificationInfo.SetFreeStyle();
						tzpManger.SetFreeStyle();
						break;
					default:
						return;
					}
				}
			}
		}

		// Token: 0x0600018A RID: 394 RVA: 0x00006F64 File Offset: 0x00005164
		public static void CanExecuteSetFreeStyle(object sender, CanExecuteRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			e.CanExecute = false;
			if (@this.ActiveDocument != null)
			{
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				if (tzpManger == null)
				{
					return;
				}
				switch (tzpManger.Type)
				{
				case TzpType.ReportCode:
				case TzpType.Code:
					e.CanExecute = tzpManger.IsNormalStyle && tzpManger.Booking && (tzpManger.ProgType == "M" || tzpManger.ProgType == "S");
					return;
				case TzpType.Form:
					e.CanExecute = false;
					return;
				}
				e.CanExecute = false;
			}
		}

		// Token: 0x1700006F RID: 111
		// (get) Token: 0x0600018B RID: 395 RVA: 0x00007014 File Offset: 0x00005214
		public static RoutedCommand ChangeProgramTemplateCommand
		{
			get
			{
				if (MenuCommands._changeProgramTemplateCommand == null)
				{
					MenuCommands._changeProgramTemplateCommand = new RoutedCommand("ChangeProgramTemplateCommand", typeof(MenuCommands));
				}
				return MenuCommands._changeProgramTemplateCommand;
			}
		}

		// Token: 0x0600018C RID: 396 RVA: 0x0000703C File Offset: 0x0000523C
		public static void ExecutedChangeProgramTemplate(object sender, ExecutedRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			if (@this.ActiveDocument != null)
			{
				if (ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).login_user == "topstd")
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_TopstdCantChangeProgramTemplate") as string);
					return;
				}
				FileViewModel activeDocument = @this.ActiveDocument;
				SettingManager.Get().GetTzpManger(activeDocument.Key);
				CodeTemplateSelector codeTemplateSelector = new CodeTemplateSelector(activeDocument.Key);
				codeTemplateSelector.ShowDialog();
			}
		}

		// Token: 0x0600018D RID: 397 RVA: 0x000070C0 File Offset: 0x000052C0
		public static void CanChangeProgramTemplate(object sender, CanExecuteRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			e.CanExecute = false;
			if (@this.ActiveDocument != null)
			{
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				if (tzpManger == null)
				{
					return;
				}
				bool flag = tzpManger.Type == TzpType.Code || tzpManger.Type == TzpType.ReportCode;
				bool flag2 = !tzpManger.IsNormalStyle;
				e.CanExecute = flag && !flag2 && tzpManger.Booking;
			}
		}

		// Token: 0x17000070 RID: 112
		// (get) Token: 0x0600018E RID: 398 RVA: 0x00007135 File Offset: 0x00005335
		public static RoutedCommand ChangeStartargCommand
		{
			get
			{
				if (MenuCommands._changeStartargCommand == null)
				{
					MenuCommands._changeStartargCommand = new RoutedCommand("ChangeStartargCommand", typeof(MenuCommands));
				}
				return MenuCommands._changeStartargCommand;
			}
		}

		// Token: 0x0600018F RID: 399 RVA: 0x0000715C File Offset: 0x0000535C
		public static void CanExecuteChangeStartarg(object sender, CanExecuteRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			e.CanExecute = false;
			if (@this.ActiveDocument != null)
			{
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				if (tzpManger == null)
				{
					return;
				}
				bool flag = tzpManger.Type == TzpType.Code || tzpManger.Type == TzpType.ReportCode;
				e.CanExecute = flag && tzpManger.Booking;
			}
		}

		// Token: 0x06000190 RID: 400 RVA: 0x000071C4 File Offset: 0x000053C4
		public static void ExecutedChangeStartarg(object sender, ExecutedRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			if (@this.ActiveDocument != null)
			{
				if (ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).login_user == "topstd")
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_TopstdCantChangeStartarg") as string);
					return;
				}
				FileViewModel activeDocument = @this.ActiveDocument;
				SettingManager.Get().GetTzpManger(activeDocument.Key);
				StartArgSetting startArgSetting = new StartArgSetting(activeDocument.Key);
				startArgSetting.ShowDialog();
			}
		}

		// Token: 0x17000071 RID: 113
		// (get) Token: 0x06000191 RID: 401 RVA: 0x00007245 File Offset: 0x00005445
		public static RoutedCommand adzp188Command
		{
			get
			{
				if (MenuCommands._adzq188Command == null)
				{
					MenuCommands._adzq188Command = new RoutedCommand("adzq188Command", typeof(MenuCommands));
				}
				return MenuCommands._adzq188Command;
			}
		}

		// Token: 0x06000192 RID: 402 RVA: 0x0000726C File Offset: 0x0000546C
		public static void Executedaadzp188(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.RunProgram("adzp188");
		}

		// Token: 0x17000072 RID: 114
		// (get) Token: 0x06000193 RID: 403 RVA: 0x00007279 File Offset: 0x00005479
		public static RoutedCommand azzi301Command
		{
			get
			{
				if (MenuCommands._azzi301Command == null)
				{
					MenuCommands._azzi301Command = new RoutedCommand("azzi301Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi301Command;
			}
		}

		// Token: 0x06000194 RID: 404 RVA: 0x000072A0 File Offset: 0x000054A0
		public static void Executedazzi301(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.RunProgram("azzi301");
		}

		// Token: 0x17000073 RID: 115
		// (get) Token: 0x06000195 RID: 405 RVA: 0x000072B4 File Offset: 0x000054B4
		public static RoutedCommand azzi300Command
		{
			get
			{
				if (MenuCommands._azzi300Command == null)
				{
					MenuCommands._azzi300Command = new RoutedCommand("azzi300Command", typeof(MenuCommands));
				}
				return MenuCommands._azzi300Command;
			}
		}

		// Token: 0x06000196 RID: 406 RVA: 0x000072DB File Offset: 0x000054DB
		public static void Executedazzi300(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.RunProgram("azzi300");
		}

		// Token: 0x17000074 RID: 116
		// (get) Token: 0x06000197 RID: 407 RVA: 0x000072EF File Offset: 0x000054EF
		public static RoutedCommand adzp600Command
		{
			get
			{
				if (MenuCommands._adzp600Command == null)
				{
					MenuCommands._adzp600Command = new RoutedCommand("adzp600Command", typeof(MenuCommands));
				}
				return MenuCommands._adzp600Command;
			}
		}

		// Token: 0x06000198 RID: 408 RVA: 0x00007316 File Offset: 0x00005516
		public static void Executedadzp600(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.RunProgram("adzp600");
		}

		// Token: 0x17000075 RID: 117
		// (get) Token: 0x06000199 RID: 409 RVA: 0x0000732A File Offset: 0x0000552A
		public static RoutedCommand ViewProgramInfomationCommand
		{
			get
			{
				if (MenuCommands._viewProgramInfomationCommand == null)
				{
					MenuCommands._viewProgramInfomationCommand = new RoutedCommand("ViewProgramInfomationCommand", typeof(MenuCommands));
				}
				return MenuCommands._viewProgramInfomationCommand;
			}
		}

		// Token: 0x0600019A RID: 410 RVA: 0x00007354 File Offset: 0x00005554
		public static void ExecutedViewProgramInfomation(object sender, ExecutedRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			if (@this.ActiveDocument != null)
			{
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				XElement xelement = null;
				switch (tzpManger.Type)
				{
				case TzpType.ReportSpec:
					xelement = tzpManger.ReportSpecificationInfo.ToXElement();
					break;
				case TzpType.ReportCode:
				case TzpType.Code:
					xelement = ResourceController.GetInstance().GetProgramInfo(activeDocument.Key).TAP;
					break;
				case TzpType.CodeSpec:
					xelement = tzpManger.FglSpecificationInfo.Source;
					break;
				case TzpType.Form:
					xelement = tzpManger.SpecificationInfo.TSDElement;
					break;
				}
				if (xelement != null)
				{
					ProgramInfomationWindow programInfomationWindow = new ProgramInfomationWindow();
					programInfomationWindow.Load(activeDocument.Key, xelement);
					programInfomationWindow.ShowDialog();
				}
			}
		}

		// Token: 0x0600019B RID: 411 RVA: 0x0000741C File Offset: 0x0000561C
		public static void CanExecuteViewProgramInfomation(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			EditorWorkspace @this = EditorWorkspace.This;
			bool flag;
			if (e.Parameter == null || null == MenuCommands.ProgramKey)
			{
				flag = false;
			}
			else if (@this.ActiveDocument == null)
			{
				flag = false;
			}
			else
			{
				switch (MenuCommands.ProgramKey.PackType)
				{
				case TzpType.ReportSpec:
				case TzpType.CodeSpec:
				case TzpType.Form:
					flag = string.Equals("Spec", e.Parameter.ToString(), StringComparison.CurrentCultureIgnoreCase);
					break;
				case TzpType.ReportCode:
				case TzpType.Code:
					flag = string.Equals("Code", e.Parameter.ToString(), StringComparison.CurrentCultureIgnoreCase);
					break;
				default:
					flag = false;
					break;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x17000076 RID: 118
		// (get) Token: 0x0600019C RID: 412 RVA: 0x000074D5 File Offset: 0x000056D5
		public static RoutedCommand ToggleDiffEditableCommand
		{
			get
			{
				if (MenuCommands._toggleDiffEditableCommand == null)
				{
					MenuCommands._toggleDiffEditableCommand = new RoutedCommand("ToggleDiffEditableCommand", typeof(MenuCommands));
				}
				return MenuCommands._toggleDiffEditableCommand;
			}
		}

		// Token: 0x0600019D RID: 413 RVA: 0x000074FC File Offset: 0x000056FC
		public static void ExecutedToggleDiffEditable(object sender, ExecutedRoutedEventArgs e)
		{
			FileViewModel activeDocument = EditorWorkspace.This.ActiveDocument;
			EventAggregatorManager.Global.GetEvent<ToggleDiffEditableEvent>().Publish(activeDocument.Key);
		}

		// Token: 0x0600019E RID: 414 RVA: 0x0000752C File Offset: 0x0000572C
		public static void CanExecuteToggleDiffEditable(object sender, CanExecuteRoutedEventArgs e)
		{
			FileViewModel activeDocument = EditorWorkspace.This.ActiveDocument;
			TzpManager tzpManager = ((activeDocument == null) ? null : SettingManager.Get().GetTzpManger(activeDocument.Key));
			e.CanExecute = tzpManager != null && tzpManager.Type == TzpType.Code && tzpManager.IsDiff;
		}

		// Token: 0x0600019F RID: 415 RVA: 0x00007579 File Offset: 0x00005779
		private static void ForceUpdateSource()
		{
			TextBoxHelper.ForceUpdateSource();
		}

		// Token: 0x17000077 RID: 119
		// (get) Token: 0x060001A0 RID: 416 RVA: 0x00007580 File Offset: 0x00005780
		public static RoutedCommand SimpleFormSetupCommand
		{
			get
			{
				if (MenuCommands._simpleFormSetupCommand == null)
				{
					MenuCommands._simpleFormSetupCommand = new RoutedCommand("SimpleFormSetupCommand", typeof(MenuCommands));
				}
				return MenuCommands._simpleFormSetupCommand;
			}
		}

		// Token: 0x060001A1 RID: 417 RVA: 0x000075A7 File Offset: 0x000057A7
		public static void ExecutedSimpleFormSetup(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenSimpleFormFile();
		}

		// Token: 0x060001A2 RID: 418 RVA: 0x000075BA File Offset: 0x000057BA
		public static void CanExecuteSimpleFormSetup(object sender, CanExecuteRoutedEventArgs e)
		{
			if (SettingManager.Get().ErpVer == "1.0")
			{
				e.CanExecute = false;
				return;
			}
			if (SettingManager.Get().ErpVer == "3.0")
			{
				e.CanExecute = true;
			}
		}

		// Token: 0x17000078 RID: 120
		// (get) Token: 0x060001A3 RID: 419 RVA: 0x000075F7 File Offset: 0x000057F7
		public static RoutedCommand SimpleFormDownloadCommand
		{
			get
			{
				if (MenuCommands._simpleFormDownloadCommand == null)
				{
					MenuCommands._simpleFormDownloadCommand = new RoutedCommand("SimpleFormDownloadCommand", typeof(MenuCommands));
				}
				return MenuCommands._simpleFormDownloadCommand;
			}
		}

		// Token: 0x060001A4 RID: 420 RVA: 0x0000761E File Offset: 0x0000581E
		public static void ExecutedSimpleFormDownload(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.SimpleFormDownload(SettingManager.Get().CurrentSetting.Connection.Workspace);
		}

		// Token: 0x060001A5 RID: 421 RVA: 0x00007641 File Offset: 0x00005841
		public static void CanExecuteSimpleFormDownload(object sender, CanExecuteRoutedEventArgs e)
		{
			if (SettingManager.Get().ErpVer == "1.0")
			{
				e.CanExecute = false;
				return;
			}
			if (SettingManager.Get().ErpVer == "3.0")
			{
				e.CanExecute = true;
			}
		}

		// Token: 0x17000079 RID: 121
		// (get) Token: 0x060001A6 RID: 422 RVA: 0x0000767E File Offset: 0x0000587E
		public static RoutedCommand SimpleFormUploadCommand
		{
			get
			{
				if (MenuCommands._simpleFormUploadCommand == null)
				{
					MenuCommands._simpleFormUploadCommand = new RoutedCommand("SimpleFormUploadCommand", typeof(MenuCommands));
				}
				return MenuCommands._simpleFormUploadCommand;
			}
		}

		// Token: 0x060001A7 RID: 423 RVA: 0x000076A8 File Offset: 0x000058A8
		public static void ExecutedSimpleFormUpload(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (EditorWorkspace.This.ActiveDocument.IsModified)
			{
				MessageBoxResult messageBoxResult = ((!PreferenceManager.Current.Settings.RemaindSave) ? MessageBoxResult.Yes : DesignerMessageBox.Show(Application.Current.FindResource("Message_SaveBeforeUpload") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question));
				if (MessageBoxResult.Yes == messageBoxResult)
				{
					SettingManager.Get().SaveSetting(MenuCommands.ProgramKey);
				}
			}
			ConnectionManager.SimpleFormUpload(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "Y");
		}

		// Token: 0x060001A8 RID: 424 RVA: 0x00007744 File Offset: 0x00005944
		public static void CanExecuteSimpleFormUpload(object sender, CanExecuteRoutedEventArgs e)
		{
			FormViewModel formViewModel = EditorWorkspace.This.ActiveDocument as FormViewModel;
			if (formViewModel != null && ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x1700007A RID: 122
		// (get) Token: 0x060001A9 RID: 425 RVA: 0x00007786 File Offset: 0x00005986
		public static RoutedCommand InsertIndFunctionSetupCommand
		{
			get
			{
				if (MenuCommands._insertIndFunctionSetupCommand == null)
				{
					MenuCommands._insertIndFunctionSetupCommand = new RoutedCommand("InsertIndFunctionSetupCommand", typeof(MenuCommands));
				}
				return MenuCommands._insertIndFunctionSetupCommand;
			}
		}

		// Token: 0x060001AA RID: 426 RVA: 0x000077AD File Offset: 0x000059AD
		public static void ExecutedInsertIndFunctionSetup(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SettingManager.Get().OpenSubInsertIndFunctionFile();
		}

		// Token: 0x060001AB RID: 427 RVA: 0x000077C0 File Offset: 0x000059C0
		public static void CanExecuteInsertIndFunctionSetup(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x1700007B RID: 123
		// (get) Token: 0x060001AC RID: 428 RVA: 0x000077C9 File Offset: 0x000059C9
		public static RoutedCommand UploadIndFunctionSetupCommand
		{
			get
			{
				if (MenuCommands._uploadIndFunctionSetupCommand == null)
				{
					MenuCommands._uploadIndFunctionSetupCommand = new RoutedCommand("UploadIndFunctionSetupCommand", typeof(MenuCommands));
				}
				return MenuCommands._uploadIndFunctionSetupCommand;
			}
		}

		// Token: 0x060001AD RID: 429 RVA: 0x000077F0 File Offset: 0x000059F0
		public static void ExecutedUploadIndFunctionSetup(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (EditorWorkspace.This.ActiveDocument.IsModified)
			{
				MessageBoxResult messageBoxResult = ((!PreferenceManager.Current.Settings.RemaindSave) ? MessageBoxResult.Yes : DesignerMessageBox.Show(Application.Current.FindResource("Message_SaveBeforeUpload") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question));
				if (MessageBoxResult.Yes == messageBoxResult)
				{
					SettingManager.Get().SaveSetting(MenuCommands.ProgramKey);
				}
			}
			ConnectionManager.UploadIndFunctionSetup(Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ZipFile);
		}

		// Token: 0x060001AE RID: 430 RVA: 0x00007884 File Offset: 0x00005A84
		public static void CanExecuteUploaIndFunctionSetup(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = false;
			if (MenuCommands.ProgramKey != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).isIndFun)
			{
				e.CanExecute = true;
			}
		}

		// Token: 0x1700007C RID: 124
		// (get) Token: 0x060001AF RID: 431 RVA: 0x000078D3 File Offset: 0x00005AD3
		public static RoutedCommand PreferenceShortcut
		{
			get
			{
				if (MenuCommands._preferenceShortcut == null)
				{
					MenuCommands._preferenceShortcut = new RoutedCommand("PreferenceShortcut", typeof(MenuCommands));
				}
				return MenuCommands._preferenceShortcut;
			}
		}

		// Token: 0x060001B0 RID: 432 RVA: 0x000078FA File Offset: 0x00005AFA
		public static void ExecutedPreferenceShortcut(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ConnectionManager.RunProgram(e.Parameter.ToString());
		}

		// Token: 0x1700007D RID: 125
		// (get) Token: 0x060001B1 RID: 433 RVA: 0x00007914 File Offset: 0x00005B14
		public static RoutedCommand ProgramThisVersionModifyCommand
		{
			get
			{
				if (MenuCommands._programThisVersionModifyCommand == null)
				{
					MenuCommands._programThisVersionModifyCommand = new RoutedCommand("ProgramThisVersionModifyCommand", typeof(MenuCommands));
				}
				return MenuCommands._programThisVersionModifyCommand;
			}
		}

		// Token: 0x060001B2 RID: 434 RVA: 0x0000793C File Offset: 0x00005B3C
		public static void CanExecuteProgramThisVersionModify(object sender, CanExecuteRoutedEventArgs e)
		{
			if (!(MenuCommands.ProgramKey != null) || (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type != TzpType.Code && SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type != TzpType.ReportCode))
			{
				e.CanExecute = false;
				return;
			}
			if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060001B3 RID: 435 RVA: 0x000079B4 File Offset: 0x00005BB4
		public static void ExecutedProgramThisVersionModify(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ThisVersionModifyConfirm") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question);
			if (messageBoxResult != MessageBoxResult.Yes)
			{
				return;
			}
			MenuCommands.UploadBasedOnType_Modify(SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type);
		}

		// Token: 0x1700007E RID: 126
		// (get) Token: 0x060001B4 RID: 436 RVA: 0x00007A17 File Offset: 0x00005C17
		public static RoutedCommand SpecThisVersionModifyCommand
		{
			get
			{
				if (MenuCommands._specThisVersionModifyCommand == null)
				{
					MenuCommands._specThisVersionModifyCommand = new RoutedCommand("SpecThisVersionModifyCommand", typeof(MenuCommands));
				}
				return MenuCommands._specThisVersionModifyCommand;
			}
		}

		// Token: 0x060001B5 RID: 437 RVA: 0x00007A40 File Offset: 0x00005C40
		public static void CanExecuteSpecThisVersionModify(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (ManagedForm.Current != null && ManagedForm.Current.IsSimpleForm)
			{
				e.CanExecute = false;
				return;
			}
			if (null == MenuCommands.ProgramKey)
			{
				flag = false;
			}
			else if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey) != null)
			{
				if (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Booking)
				{
					switch (SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type)
					{
					case TzpType.ReportSpec:
					case TzpType.CodeSpec:
					case TzpType.Form:
						flag = true;
						goto IL_0092;
					}
					flag = false;
				}
				else
				{
					flag = false;
				}
			}
			IL_0092:
			e.CanExecute = flag;
		}

		// Token: 0x060001B6 RID: 438 RVA: 0x00007AE8 File Offset: 0x00005CE8
		public static void ExecutedSpecThisVersionModify(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ThisVersionModifyConfirm") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo, MessageBoxImage.Question);
			if (messageBoxResult != MessageBoxResult.Yes)
			{
				return;
			}
			MenuCommands.UploadBasedOnType_Modify(SettingManager.Get().GetTzpManger(MenuCommands.ProgramKey).Type);
		}

		// Token: 0x1700007F RID: 127
		// (get) Token: 0x060001B7 RID: 439 RVA: 0x00007B4B File Offset: 0x00005D4B
		public static RoutedCommand SpecAdjustContainerBlankAreaCommand
		{
			get
			{
				if (MenuCommands._specAdjustContainerBlankAreaCommand == null)
				{
					MenuCommands._specAdjustContainerBlankAreaCommand = new RoutedCommand("SpecAdjustContainerBlankAreaCommand", typeof(MenuCommands));
				}
				return MenuCommands._specAdjustContainerBlankAreaCommand;
			}
		}

		// Token: 0x060001B8 RID: 440 RVA: 0x00007B74 File Offset: 0x00005D74
		public static void CanExecuteSpecAdjustContainerBlankArea(object sender, CanExecuteRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			bool flag;
			if (e.Parameter == null || null == MenuCommands.ProgramKey)
			{
				flag = false;
			}
			else if (@this.ActiveDocument == null)
			{
				flag = false;
			}
			else
			{
				switch (MenuCommands.ProgramKey.PackType)
				{
				case TzpType.ReportSpec:
				case TzpType.CodeSpec:
				case TzpType.Form:
					flag = string.Equals("Spec", e.Parameter.ToString(), StringComparison.CurrentCultureIgnoreCase);
					break;
				case TzpType.ReportCode:
				case TzpType.Code:
					flag = string.Equals("Code", e.Parameter.ToString(), StringComparison.CurrentCultureIgnoreCase);
					break;
				default:
					flag = false;
					break;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060001B9 RID: 441 RVA: 0x00007C14 File Offset: 0x00005E14
		public static void ExecutedSpecAdjustContainerBlankArea(object sender, ExecutedRoutedEventArgs e)
		{
			EditorWorkspace @this = EditorWorkspace.This;
			if (@this.ActiveDocument != null)
			{
				FileViewModel activeDocument = @this.ActiveDocument;
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(activeDocument.Key);
				if (MessageBox.Show(Application.Current.FindResource("Message_AdjustContainerBlankArea") as string, Application.Current.FindResource("Button_OK") as string, MessageBoxButton.OKCancel, MessageBoxImage.Question) == MessageBoxResult.OK)
				{
					MenuCommands.ShrinkContainer(tzpManger.SpecificationInfo.FormNode);
					tzpManger.SpecificationInfo.FormNode["gridHeight"] = "1";
					tzpManger.SpecificationInfo.FormNode["gridWidth"] = "1";
				}
			}
		}

		// Token: 0x060001BA RID: 442 RVA: 0x00007CC4 File Offset: 0x00005EC4
		private static void ShrinkContainer(XmlElement formNode)
		{
			foreach (XmlElement xmlElement in formNode.Nodes)
			{
				if (!(xmlElement.Name == "worksheet") && (xmlElement.Type == ComponentType.Grid || xmlElement.Type == ComponentType.HBox || xmlElement.Type == ComponentType.VBox || xmlElement.Type == ComponentType.Group || xmlElement.Type == ComponentType.Folder || xmlElement.Type == ComponentType.Page))
				{
					MenuCommands.ShrinkContainer(xmlElement);
					xmlElement["gridHeight"] = "1";
					xmlElement["gridWidth"] = "1";
				}
			}
		}

		// Token: 0x060001BB RID: 443 RVA: 0x00007D7C File Offset: 0x00005F7C
		private static void UploadBasedOnType_Modify(TzpType type)
		{
			switch (type)
			{
			case TzpType.ReportSpec:
				MenuCommands.UpdateReportSpecification_Modify();
				return;
			case TzpType.ReportCode:
			case TzpType.Code:
				MenuCommands.UpdateProgram_Modify(type);
				return;
			case TzpType.CodeSpec:
				MenuCommands.UpdateCodeSpecification_Modify();
				return;
			case TzpType.Form:
				MenuCommands.UpdateSpecification_Modify();
				return;
			default:
				return;
			}
		}

		// Token: 0x060001BC RID: 444 RVA: 0x00007DC0 File Offset: 0x00005FC0
		private static void UpdateProgram_Modify(TzpType type)
		{
			string value = ResourceController.GetInstance().GetProgramInfo(MenuCommands.ProgramKey).TAP.Attribute("type").Value;
			if (value.Equals("G", StringComparison.CurrentCultureIgnoreCase))
			{
				ConnectionManager.UploadCode_Modify(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, TzpType.ReportCode.Description());
				return;
			}
			ConnectionManager.UploadCode_Modify(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, TzpType.Code.Description());
		}

		// Token: 0x060001BD RID: 445 RVA: 0x00007E4F File Offset: 0x0000604F
		private static void UpdateCodeSpecification_Modify()
		{
			ConnectionManager.UploadSpec_Modify(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "CSPEC");
		}

		// Token: 0x060001BE RID: 446 RVA: 0x00007E70 File Offset: 0x00006070
		private static void UpdateReportSpecification_Modify()
		{
			ConnectionManager.UploadSpec_Modify(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "RSPEC");
		}

		// Token: 0x060001BF RID: 447 RVA: 0x00007E91 File Offset: 0x00006091
		private static void UpdateSpecification_Modify()
		{
			ConnectionManager.UploadSpec_Modify(MenuCommands.ZipFile, Path.GetFileName(MenuCommands.ZipFile), MenuCommands.ModuleName, "SPEC");
		}

		// Token: 0x0400004D RID: 77
		private static RoutedCommand _fileOpenCommand;

		// Token: 0x0400004E RID: 78
		private static RoutedCommand _specOpenCommand;

		// Token: 0x0400004F RID: 79
		private static RoutedCommand _fileCloseCommand;

		// Token: 0x04000050 RID: 80
		private static RoutedCommand _fileCloseAllCommand;

		// Token: 0x04000051 RID: 81
		private static RoutedCommand _exitCommand;

		// Token: 0x04000052 RID: 82
		private static RoutedCommand _diffCommand;

		// Token: 0x04000053 RID: 83
		private static RoutedCommand _showAboutCommand;

		// Token: 0x04000054 RID: 84
		private static RoutedCommand _showShorcutListCommand;

		// Token: 0x04000055 RID: 85
		private static RoutedCommand _showViCommandListCommand;

		// Token: 0x04000056 RID: 86
		private static RoutedCommand _baseDataViewCommand;

		// Token: 0x04000057 RID: 87
		private static RoutedCommand _tableViewCommand;

		// Token: 0x04000058 RID: 88
		private static RoutedCommand _columnViewCommand;

		// Token: 0x04000059 RID: 89
		private static RoutedCommand _basicDataUpdateCommand;

		// Token: 0x0400005A RID: 90
		private static RoutedCommand _selectbasicDataUpdateCommand;

		// Token: 0x0400005B RID: 91
		private static RoutedCommand _regenerateBasicDataCommand;

		// Token: 0x0400005C RID: 92
		private static RoutedCommand _specificationViewCommand;

		// Token: 0x0400005D RID: 93
		private static RoutedCommand _uncitedSpecificationViewCommand;

		// Token: 0x0400005E RID: 94
		private static RoutedCommand _specificationUploadResetTAPCommand;

		// Token: 0x0400005F RID: 95
		private static RoutedCommand _specificationDownloadCommand;

		// Token: 0x04000060 RID: 96
		private static RoutedCommand _programDownloadCommand;

		// Token: 0x04000061 RID: 97
		private static RoutedCommand _programDebugCommand;

		// Token: 0x04000062 RID: 98
		private static RoutedCommand _programErrorCheckCommand;

		// Token: 0x04000063 RID: 99
		private static RoutedCommand _programTestCommand;

		// Token: 0x04000064 RID: 100
		private static RoutedCommand _programRegenCommand;

		// Token: 0x04000065 RID: 101
		private static RoutedCommand _showLogCommand;

		// Token: 0x04000066 RID: 102
		private static RoutedCommand _showUncitedCommand;

		// Token: 0x04000067 RID: 103
		private static RoutedCommand _showSectionUncitedCommand;

		// Token: 0x04000068 RID: 104
		private static RoutedCommand _showCustomAdpListCommand;

		// Token: 0x04000069 RID: 105
		private static RoutedCommand _siteManagerCommand;

		// Token: 0x0400006A RID: 106
		private static RoutedCommand _changeLayoutCommand;

		// Token: 0x0400006B RID: 107
		private static RoutedCommand _findCommand;

		// Token: 0x0400006C RID: 108
		private static RoutedCommand _replaceCommand;

		// Token: 0x0400006D RID: 109
		public static RoutedCommand _exportDocxCommand;

		// Token: 0x0400006E RID: 110
		private static RoutedCommand _specCodeUploadCommand;

		// Token: 0x0400006F RID: 111
		private static RoutedCommand _specCodeUploadAndRC3Command;

		// Token: 0x04000070 RID: 112
		private static RoutedCommand _precompileCommand;

		// Token: 0x04000071 RID: 113
		public static RoutedCommand _adzi140Command;

		// Token: 0x04000072 RID: 114
		public static RoutedCommand _commonUsedMenuCommand;

		// Token: 0x04000073 RID: 115
		public static RoutedCommand _adzi150Command;

		// Token: 0x04000074 RID: 116
		public static RoutedCommand _adzi210Command;

		// Token: 0x04000075 RID: 117
		public static RoutedCommand _adzi220Command;

		// Token: 0x04000076 RID: 118
		public static RoutedCommand _adzi400Command;

		// Token: 0x04000077 RID: 119
		public static RoutedCommand _adzq490Command;

		// Token: 0x04000078 RID: 120
		public static RoutedCommand _azzi600Command;

		// Token: 0x04000079 RID: 121
		public static RoutedCommand _azzi650Command;

		// Token: 0x0400007A RID: 122
		public static RoutedCommand _azzi900Command;

		// Token: 0x0400007B RID: 123
		public static RoutedCommand _azzi901Command;

		// Token: 0x0400007C RID: 124
		public static RoutedCommand _azzi910Command;

		// Token: 0x0400007D RID: 125
		public static RoutedCommand _azzi920Command;

		// Token: 0x0400007E RID: 126
		public static RoutedCommand _adzp168Command;

		// Token: 0x0400007F RID: 127
		public static RoutedCommand _adzp165Command;

		// Token: 0x04000080 RID: 128
		public static RoutedCommand _adzq255Command;

		// Token: 0x04000081 RID: 129
		public static RoutedCommand _reportDownloadCommand;

		// Token: 0x04000082 RID: 130
		public static RoutedCommand _reportUploadCommand;

		// Token: 0x04000083 RID: 131
		public static RoutedCommand _reportOpenCommand;

		// Token: 0x04000084 RID: 132
		public static RoutedCommand _adzp270Command;

		// Token: 0x04000085 RID: 133
		public static RoutedCommand _preferenceWindowCommand;

		// Token: 0x04000086 RID: 134
		public static RoutedCommand _recentFilesCommand;

		// Token: 0x04000087 RID: 135
		public static RoutedCommand _adzp990ExpCommand;

		// Token: 0x04000088 RID: 136
		public static RoutedCommand _adzp990ImpCommand;

		// Token: 0x04000089 RID: 137
		public static RoutedCommand _adzi888ExpCommand;

		// Token: 0x0400008A RID: 138
		public static RoutedCommand _adzi888ImpCommand;

		// Token: 0x0400008B RID: 139
		public static RoutedCommand _azzi909Command;

		// Token: 0x0400008C RID: 140
		public static RoutedCommand _formPreviewCommand;

		// Token: 0x0400008D RID: 141
		private static RoutedCommand _showGeneroUserGuideCommand;

		// Token: 0x0400008E RID: 142
		private static RoutedCommand _showTOPSTDLoginCommand;

		// Token: 0x0400008F RID: 143
		public static RoutedCommand _specCheckInCommand;

		// Token: 0x04000090 RID: 144
		public static RoutedCommand _adzq001Command;

		// Token: 0x04000091 RID: 145
		public static RoutedCommand _adzq003Command;

		// Token: 0x04000092 RID: 146
		private static RoutedCommand _fileOpenSpecCommand;

		// Token: 0x04000093 RID: 147
		private static RoutedCommand _fileOpenCodeCommand;

		// Token: 0x04000094 RID: 148
		private static RoutedCommand _showServiceCloudLoginCommand;

		// Token: 0x04000095 RID: 149
		public static RoutedCommand _toggleLayoutCommand;

		// Token: 0x04000096 RID: 150
		public static RoutedCommand _toggleLiteCommand;

		// Token: 0x04000097 RID: 151
		private static RoutedCommand _viewFormDiffListCommand;

		// Token: 0x04000098 RID: 152
		private static RoutedCommand _nextdiffCommand;

		// Token: 0x04000099 RID: 153
		private static RoutedCommand _searchKeySettingWindowCommand;

		// Token: 0x0400009A RID: 154
		private static EventHandler MenuCommands_TzpSaved = delegate(object sender, EventArgs e)
		{
		};

		// Token: 0x0400009B RID: 155
		private static RoutedCommand _srRelationSettingCommand;

		// Token: 0x0400009C RID: 156
		public static RoutedCommand _setFreeStyleCommand;

		// Token: 0x0400009D RID: 157
		public static RoutedCommand _changeProgramTemplateCommand;

		// Token: 0x0400009E RID: 158
		public static RoutedCommand _changeStartargCommand;

		// Token: 0x0400009F RID: 159
		public static RoutedCommand _adzq188Command;

		// Token: 0x040000A0 RID: 160
		public static RoutedCommand _azzi301Command;

		// Token: 0x040000A1 RID: 161
		public static RoutedCommand _azzi300Command;

		// Token: 0x040000A2 RID: 162
		public static RoutedCommand _adzp600Command;

		// Token: 0x040000A3 RID: 163
		public static RoutedCommand _viewProgramInfomationCommand;

		// Token: 0x040000A4 RID: 164
		private static RoutedCommand _toggleDiffEditableCommand;

		// Token: 0x040000A5 RID: 165
		private static RoutedCommand _simpleFormSetupCommand = null;

		// Token: 0x040000A6 RID: 166
		private static RoutedCommand _simpleFormDownloadCommand = null;

		// Token: 0x040000A7 RID: 167
		private static RoutedCommand _simpleFormUploadCommand = null;

		// Token: 0x040000A8 RID: 168
		private static RoutedCommand _insertIndFunctionSetupCommand = null;

		// Token: 0x040000A9 RID: 169
		private static RoutedCommand _uploadIndFunctionSetupCommand = null;

		// Token: 0x040000AA RID: 170
		public static RoutedCommand _preferenceShortcut;

		// Token: 0x040000AB RID: 171
		private static RoutedCommand _programThisVersionModifyCommand;

		// Token: 0x040000AC RID: 172
		private static RoutedCommand _specThisVersionModifyCommand;

		// Token: 0x040000AD RID: 173
		private static RoutedCommand _specAdjustContainerBlankAreaCommand;
	}
}
