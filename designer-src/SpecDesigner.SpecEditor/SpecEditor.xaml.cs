using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000040 RID: 64
	public partial class SpecEditor : UserControl, INotifyPropertyChanged, IDisposable
	{
		// Token: 0x06000196 RID: 406 RVA: 0x0000B2F0 File Offset: 0x000094F0
		public SpecEditor(PackageKey key)
		{
			this.InitializeComponent();
			this.ProgramKey = key;
			EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Subscribe(new Action<SpecNodeLoadedEventArgs>(this.SubscribeSpecNodeLoaded));
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Subscribe(new Action<SpecArgs>(this.SubscribeHideSpecProperties));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.SubscribeTzpFileClosed));
			this.SpecEditorGrid.IsEnabled = false;
			this.init();
			base.CommandBindings.Add(new CommandBinding(MenuCommands.SetCiteCommand, new ExecutedRoutedEventHandler(this.ExecutedSetCite), new CanExecuteRoutedEventHandler(MenuCommands.CanExecuteSetCite)));
		}

		// Token: 0x06000197 RID: 407 RVA: 0x0000B3A4 File Offset: 0x000095A4
		public void SubscribeHideSpecProperties(SpecArgs s)
		{
			if (s.ProgramKey != this.ProgramKey)
			{
				return;
			}
			base.DataContext = Binding.DoNothing;
			this.SASpecTextBox.Text = "";
			this.SpecType = "";
		}

		// Token: 0x06000198 RID: 408 RVA: 0x0000B3E0 File Offset: 0x000095E0
		public void init()
		{
			this.SpecEditorGrid.IsEnabled = this.ProgramKey != null && SettingManager.Get().GetTzpManger(this.ProgramKey).Type == TzpType.Form;
			if (this.ProgramKey != null)
			{
				if (SettingManager.Get().GetTzpManger(this.ProgramKey).Type != TzpType.Form)
				{
					this.SetSASpecViewerAndCheckBoxHidden(true);
					this.SASpecTextBox.Text = "";
					return;
				}
				XElement xelement = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.TSDElement.Element("sa_spec");
				if (xelement != null)
				{
					this.SetSASpecViewerAndCheckBoxHidden(false);
					this.SASpecTextBox.Text = "";
				}
				else
				{
					this.SetSASpecViewerAndCheckBoxHidden(true);
					this.SASpecTextBox.Text = "";
				}
				this.loadProgramSpec(this.ProgramKey);
				this.loadAllSpec();
			}
		}

		// Token: 0x06000199 RID: 409 RVA: 0x0000B4D4 File Offset: 0x000096D4
		private void loadProgramSpec(PackageKey programKey)
		{
			if (null == programKey)
			{
				return;
			}
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(programKey).SpecificationInfo;
			if (specificationInfo == null)
			{
				return;
			}
			this.allButton.DataContext = specificationInfo.ProgramSpec;
			this.miallButton.DataContext = specificationInfo.ProgramMISpec;
			this.diallButton.DataContext = specificationInfo.ProgramDISpec;
			this.dballButton.DataContext = specificationInfo.ProgramDBSpec;
		}

		// Token: 0x0600019A RID: 410 RVA: 0x0000B544 File Offset: 0x00009744
		private void showOrHideCiteCheckBox(PackageKey programKey)
		{
			if (null == programKey)
			{
				return;
			}
			this.refSpecCheckBox.Visibility = (SettingManager.Get().GetTzpManger(programKey).IsStandardProgram ? Visibility.Collapsed : Visibility.Visible);
		}

		// Token: 0x0600019B RID: 411 RVA: 0x0000B574 File Offset: 0x00009774
		private void SubscribeTzpFileClosed(PackageKey programKey)
		{
			if (SettingManager.Get().GetTzpManger(programKey).Type != TzpType.Form)
			{
				return;
			}
			if (programKey != this.ProgramKey)
			{
				return;
			}
			if (base.DataContext is AbstractSpecNode)
			{
				AbstractSpecNode abstractSpecNode = base.DataContext as AbstractSpecNode;
				if (abstractSpecNode.ProgramKey.Equals(programKey))
				{
					base.DataContext = Binding.DoNothing;
				}
			}
			if (string.IsNullOrEmpty(Application.Current.MainWindow.Tag as string))
			{
				this.SpecEditorGrid.IsEnabled = false;
			}
			this.SpecType = "";
			this.Dispose();
		}

		// Token: 0x0600019C RID: 412 RVA: 0x0000B610 File Offset: 0x00009810
		public void SubscribeSpecNodeLoaded(SpecNodeLoadedEventArgs args)
		{
			base.DataContext = Binding.DoNothing;
			this.SpecEditorGrid.IsEnabled = false;
			if (args == null)
			{
				return;
			}
			if (args.ProgramKey != this.ProgramKey)
			{
				return;
			}
			AbstractSpecNode specNode = args.SpecNode;
			this.SpecType = "FIELD";
			if (specNode == null)
			{
				return;
			}
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(specNode.ProgramKey).SpecificationInfo;
			if (!(specNode is SpecActionNode) && !(specNode is SpecMultiLangNode) && !SpecificationInfo.AllowSaveWidgets.Contains(specNode.Widget))
			{
				return;
			}
			this.SpecEditorGrid.IsEnabled = true;
			base.DataContext = specNode;
			this.refSpecCheckBox.DataContext = specNode;
			this.SASpecTextBox.Text = specNode.SASpecText;
			this.showOrHideCiteCheckBox(null);
		}

		// Token: 0x0600019D RID: 413 RVA: 0x0000B6D2 File Offset: 0x000098D2
		private void SetSASpecViewerAndCheckBoxHidden(bool hide)
		{
			if (hide)
			{
				this.saSpecViewCB.Visibility = Visibility.Collapsed;
			}
			else
			{
				this.saSpecViewCB.Visibility = Visibility.Visible;
			}
			this.SetSASpecViewerHidden(hide);
		}

		// Token: 0x0600019E RID: 414 RVA: 0x0000B6F8 File Offset: 0x000098F8
		private void SetSASpecViewerHidden(bool hide)
		{
			bool flag = this.saSpecViewCB.IsChecked == true;
			if (flag && !hide)
			{
				this.SASpecViewer.Visibility = Visibility.Visible;
				this.specGrid.ColumnDefinitions[0].Width = new GridLength(0.5, GridUnitType.Star);
				this.specGrid.ColumnDefinitions[1].Width = new GridLength(0.5, GridUnitType.Star);
				return;
			}
			this.SASpecViewer.Visibility = Visibility.Collapsed;
			this.specGrid.ColumnDefinitions[0].Width = new GridLength(1.0, GridUnitType.Star);
			this.specGrid.ColumnDefinitions[1].Width = new GridLength(0.0, GridUnitType.Auto);
		}

		// Token: 0x0600019F RID: 415 RVA: 0x0000B7D8 File Offset: 0x000099D8
		private void miallButton_Click(object sender, RoutedEventArgs e)
		{
			SpecProgramMI specProgramMI = (sender as Button).DataContext as SpecProgramMI;
			if (specProgramMI == null)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Publish(new SpecArgs(null, ComponentType.Unknown, this.ProgramKey));
			this.SpecEditorGrid.IsEnabled = true;
			base.DataContext = specProgramMI;
			this.specTextBox.Focus();
			this.SpecType = "MI";
			this.refSpecCheckBox.DataContext = (sender as Button).DataContext;
			this.showOrHideCiteCheckBox(specProgramMI.ProgramKey);
		}

		// Token: 0x060001A0 RID: 416 RVA: 0x0000B864 File Offset: 0x00009A64
		private void dballButton_Click(object sender, RoutedEventArgs e)
		{
			SpecProgramDB specProgramDB = (sender as Button).DataContext as SpecProgramDB;
			if (specProgramDB == null)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Publish(new SpecArgs(null, ComponentType.Unknown, this.ProgramKey));
			this.SpecEditorGrid.IsEnabled = true;
			base.DataContext = specProgramDB;
			this.specTextBox.Focus();
			this.SpecType = "DB";
			this.refSpecCheckBox.DataContext = (sender as Button).DataContext;
			this.showOrHideCiteCheckBox(specProgramDB.ProgramKey);
		}

		// Token: 0x060001A1 RID: 417 RVA: 0x0000B8F0 File Offset: 0x00009AF0
		private void diallButton_Click(object sender, RoutedEventArgs e)
		{
			SpecProgramDI specProgramDI = (sender as Button).DataContext as SpecProgramDI;
			if (specProgramDI == null)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Publish(new SpecArgs(null, ComponentType.Unknown, this.ProgramKey));
			this.SpecEditorGrid.IsEnabled = true;
			base.DataContext = specProgramDI;
			this.specTextBox.Focus();
			this.SpecType = "DI";
			this.refSpecCheckBox.DataContext = (sender as Button).DataContext;
			this.showOrHideCiteCheckBox(specProgramDI.ProgramKey);
		}

		// Token: 0x060001A2 RID: 418 RVA: 0x0000B97A File Offset: 0x00009B7A
		private void allButton_Click(object sender, RoutedEventArgs e)
		{
			this.refSpecCheckBox.DataContext = (sender as Button).DataContext;
			this.loadAllSpec();
		}

		// Token: 0x060001A3 RID: 419 RVA: 0x0000B998 File Offset: 0x00009B98
		private void loadAllSpec()
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Publish(new SpecArgs(null, ComponentType.Unknown, this.ProgramKey));
			SpecProgramAll programSpec = specificationInfo.ProgramSpec;
			XElement xelement = specificationInfo.TSDElement.Element("sa_spec");
			if (xelement != null)
			{
				XElement xelement2 = xelement.Element("sa_all");
				if (xelement2 != null)
				{
					this.SASpecTextBox.Text = xelement2.Value;
				}
			}
			this.SpecEditorGrid.IsEnabled = true;
			base.DataContext = programSpec;
			this.specTextBox.Focus();
			this.SpecType = "ALL";
			this.showOrHideCiteCheckBox(this.ProgramKey);
		}

		// Token: 0x060001A4 RID: 420 RVA: 0x0000BA58 File Offset: 0x00009C58
		private void saSpecViewCB_Click(object sender, RoutedEventArgs e)
		{
			this.SetSASpecViewerHidden((sender as CheckBox).IsChecked != true);
		}

		// Token: 0x1700002E RID: 46
		// (get) Token: 0x060001A5 RID: 421 RVA: 0x0000BA8F File Offset: 0x00009C8F
		// (set) Token: 0x060001A6 RID: 422 RVA: 0x0000BA97 File Offset: 0x00009C97
		public string SpecType
		{
			get
			{
				return this._specType;
			}
			private set
			{
				this._specType = value;
				this.NotifyPropertyChanged("SpecType");
			}
		}

		// Token: 0x14000005 RID: 5
		// (add) Token: 0x060001A7 RID: 423 RVA: 0x0000BAAC File Offset: 0x00009CAC
		// (remove) Token: 0x060001A8 RID: 424 RVA: 0x0000BAE4 File Offset: 0x00009CE4
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060001A9 RID: 425 RVA: 0x0000BB19 File Offset: 0x00009D19
		public void NotifyPropertyChanged(string name)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(name));
			}
		}

		// Token: 0x060001AA RID: 426 RVA: 0x0000BB38 File Offset: 0x00009D38
		private void refSpecCheckBox_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			SpecProgram specProgram = (sender as CheckBox).DataContext as SpecProgram;
			bool flag = (sender as CheckBox).IsChecked != null && (sender as CheckBox).IsChecked.Value;
			if (specProgram != null)
			{
				string text = ((specProgram.Source.Name.LocalName == "all") ? "specEditor_AllSpec" : ((specProgram.Source.Name.LocalName == "mi_all") ? "specEditor_MiAllSpec" : ((specProgram.Source.Name.LocalName == "di_all") ? "specEditor_DiAllSpec" : ((specProgram.Source.Name.LocalName == "db_all") ? "specEditor_DbAllSpec" : null))));
				text = Application.Current.FindResource(text) as string;
				if (!flag)
				{
					string text2 = Application.Current.FindResource("specEditor_confirmCopyTsd") as string;
					text2 = string.Format(text2, text);
					if (DesignerMessageBox.Show(text2, Application.Current.FindResource("specEditor_refStdSpec") as string, MessageBoxButton.YesNo) == MessageBoxResult.Yes)
					{
						specProgram.IsCited = true;
						return;
					}
				}
				else
				{
					string text2 = Application.Current.FindResource("specEditor_confirmCancelCopyTsd") as string;
					text2 = string.Format(text2, text);
					if (DesignerMessageBox.Show(text2, Application.Current.FindResource("specEditor_refStdSpec") as string, MessageBoxButton.YesNo) == MessageBoxResult.Yes)
					{
						specProgram.IsCited = false;
					}
				}
			}
		}

		// Token: 0x1700002F RID: 47
		// (get) Token: 0x060001AB RID: 427 RVA: 0x0000BCBE File Offset: 0x00009EBE
		// (set) Token: 0x060001AC RID: 428 RVA: 0x0000BCC6 File Offset: 0x00009EC6
		public PackageKey ProgramKey { get; set; }

		// Token: 0x060001AD RID: 429 RVA: 0x0000BCD0 File Offset: 0x00009ED0
		public void Dispose()
		{
			EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Unsubscribe(new Action<SpecNodeLoadedEventArgs>(this.SubscribeSpecNodeLoaded));
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Unsubscribe(new Action<SpecArgs>(this.SubscribeHideSpecProperties));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.SubscribeTzpFileClosed));
			this.allButton.DataContext = Binding.DoNothing;
			this.miallButton.DataContext = Binding.DoNothing;
			this.diallButton.DataContext = Binding.DoNothing;
			this.dballButton.DataContext = Binding.DoNothing;
			base.DataContext = Binding.DoNothing;
			this.refSpecCheckBox.DataContext = Binding.DoNothing;
			base.Visibility = Visibility.Collapsed;
		}

		// Token: 0x060001AE RID: 430 RVA: 0x0000BD90 File Offset: 0x00009F90
		public void ExecutedSetCite(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SpecProgram specProgram = e.Parameter as SpecProgram;
			if (specProgram == null)
			{
				return;
			}
			bool flag = this.refSpecCheckBox.IsChecked != null && this.refSpecCheckBox.IsChecked.Value;
			string text = ((specProgram.Source.Name.LocalName == "all") ? "specEditor_AllSpec" : ((specProgram.Source.Name.LocalName == "mi_all") ? "specEditor_MiAllSpec" : ((specProgram.Source.Name.LocalName == "di_all") ? "specEditor_DiAllSpec" : ((specProgram.Source.Name.LocalName == "db_all") ? "specEditor_DbAllSpec" : null))));
			text = Application.Current.FindResource(text) as string;
			string text2 = (flag ? (Application.Current.FindResource("specEditor_confirmCopyTsd") as string) : (Application.Current.FindResource("specEditor_confirmCancelCopyTsd") as string));
			text2 = string.Format(text2, text);
			MessageBoxResult messageBoxResult = DesignerMessageBox.Show(text2, Application.Current.FindResource("specEditor_refStdSpec") as string, MessageBoxButton.YesNo);
			if (MessageBoxResult.Yes == messageBoxResult)
			{
				if (SettingManager.Get().GetTzpManger(specProgram.ProgramKey).ProgType.Equals("S") && !SettingManager.Get().GetTzpManger(specProgram.ProgramKey).IsStandardProgram && !specProgram.IsCited)
				{
					string programName = SettingManager.Get().GetTzpManger(specProgram.ProgramKey).ProgramName;
					string stdProgramName = SettingManager.Get().GetTzpManger(specProgram.ProgramKey).StdProgramName;
					specProgram.Source.ReplaceNodes(new XCData(specProgram.CitedSpec.Value.Replace(stdProgramName, programName)));
				}
				specProgram.IsCited = !specProgram.IsCited;
				return;
			}
			this.refSpecCheckBox.IsChecked = !this.refSpecCheckBox.IsChecked;
		}

		// Token: 0x040000BB RID: 187
		private string _specType;
	}
}
