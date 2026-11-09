using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000020 RID: 32
	public partial class CodeSpecificationMainWindow : UserControl, IDisposable, IAvalonFormSpecEditorLayout
	{
		// Token: 0x17000021 RID: 33
		// (get) Token: 0x0600011F RID: 287 RVA: 0x0000B378 File Offset: 0x00009578
		// (set) Token: 0x06000120 RID: 288 RVA: 0x0000B380 File Offset: 0x00009580
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x06000121 RID: 289 RVA: 0x0000B38C File Offset: 0x0000958C
		public CodeSpecificationMainWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000122 RID: 290 RVA: 0x0000B3E0 File Offset: 0x000095E0
		public CodeSpecificationMainWindow(PackageKey key)
			: this()
		{
			this.ProgramKey = key;
			this._specBox.SetBinding(TextBox.TextProperty, new Binding("Content"));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeSpecEvent>().Subscribe(new Action<PackageKey>(this.LoadedSetting));
			base.CommandBindings.Add(new CommandBinding(CodeSpecificationCommands.DeleteCommand, new ExecutedRoutedEventHandler(this.ExecuteDelete), new CanExecuteRoutedEventHandler(this.CanDelete)));
			base.CommandBindings.Add(new CommandBinding(CodeSpecificationCommands.AddItemCommand, new ExecutedRoutedEventHandler(this.ExecuteAdd), new CanExecuteRoutedEventHandler(this.CanAdd)));
			base.CommandBindings.Add(new CommandBinding(CodeSpecificationCommands.CreateFunctionCommand, new ExecutedRoutedEventHandler(this.ExecuteCreateFunction), new CanExecuteRoutedEventHandler(this.CanCreateFunction)));
			base.CommandBindings.Add(new CommandBinding(CodeSpecificationCommands.FocusFunctionCommand, new ExecutedRoutedEventHandler(this.ExecuteFocusFunction)));
			this.componentList.SelectionChanged += this.componentList_SelectionChanged;
			base.Resources.Add("ForDBs", SettingManager.Get().ForDBs);
			base.Resources.Add("Dimensions", SettingManager.Get().Dimensions);
			List<string> list = (from t in TableColumnHelper.GetTables()
				select t.Attribute("name").Value).ToList<string>();
			list.Add(string.Empty);
			base.Resources.Add("DBs", list);
			base.AddHandler(ButtonEdit.ButtonClickEvent, new RoutedEventHandler(this.OnButtonClick));
			base.AddHandler(UIElement.PreviewLostKeyboardFocusEvent, new RoutedEventHandler(this.OnLostKeyboardFocus));
			this.notBookingWatermark.Visibility = (SettingManager.Get().GetTzpManger(this.ProgramKey).Booking ? Visibility.Collapsed : Visibility.Visible);
		}

		// Token: 0x06000123 RID: 291 RVA: 0x0000B5C8 File Offset: 0x000097C8
		private void OnLostKeyboardFocus(object sender, RoutedEventArgs e)
		{
			if (!(e.OriginalSource is TextBox))
			{
				return;
			}
			IEditableCollectionView items = this.componentList.Items;
			if (items.IsAddingNew || items.IsEditingItem)
			{
				this.componentList.CommitEdit(DataGridEditingUnit.Cell, true);
				this.componentList.CommitEdit(DataGridEditingUnit.Row, true);
			}
		}

		// Token: 0x06000124 RID: 292 RVA: 0x0000B61C File Offset: 0x0000981C
		private void OnButtonClick(object sender, RoutedEventArgs e)
		{
			LibrarySelectionWindow librarySelectionWindow = new LibrarySelectionWindow();
			if (librarySelectionWindow.ShowDialog() == true)
			{
				FrameworkElement frameworkElement = e.OriginalSource as FrameworkElement;
				if (frameworkElement != null)
				{
					FglFrontComp fglFrontComp = frameworkElement.DataContext as FglFrontComp;
					fglFrontComp.ID = librarySelectionWindow.SelectedID;
				}
			}
		}

		// Token: 0x06000125 RID: 293 RVA: 0x0000B674 File Offset: 0x00009874
		private void LoadedSetting(PackageKey key)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeSpecEvent>().Unsubscribe(new Action<PackageKey>(this.LoadedSetting));
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Subscribe(new Action<PackageKey>(this.OnSaveSetting));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.OnTzpFileClose));
			this.SpecificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).FglSpecificationInfo;
			base.DataContext = this.SpecificationInfo.FilteredComponents.View;
		}

		// Token: 0x06000126 RID: 294 RVA: 0x0000B714 File Offset: 0x00009914
		private void componentList_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.inputList.DataContext = null;
			this.returnList.DataContext = null;
			this.keywordList.DataContext = null;
			this.fcompList.DataContext = null;
			this.dimensionList.DataContext = null;
			this.testList.DataContext = null;
			FglComponent fglComponent = this.componentList.SelectedItem as FglComponent;
			if (fglComponent == null)
			{
				return;
			}
			this.inputList.DataContext = fglComponent.Inputs;
			this.returnList.DataContext = fglComponent.Returns;
			this.keywordList.DataContext = fglComponent.FglKeywords;
			this.fcompList.DataContext = fglComponent.FglFrontComps;
			this.dimensionList.DataContext = fglComponent.FglDimensions;
			this.testList.DataContext = fglComponent.FglTests;
			this._specBox.DataContext = fglComponent.SpecContent;
		}

		// Token: 0x06000127 RID: 295 RVA: 0x0000B7F8 File Offset: 0x000099F8
		private void ExecuteFocusFunction(object target, ExecutedRoutedEventArgs e)
		{
			CodeSpecificationMainWindow codeSpecificationMainWindow = target as CodeSpecificationMainWindow;
			if (codeSpecificationMainWindow != null)
			{
				FglComponent fglComponent = e.Parameter as FglComponent;
				if (fglComponent == null)
				{
					DocumentErrorsEventArgs e2 = new DocumentErrorsEventArgs();
					e2.ProgramKey = this.ProgramKey;
					e2.SourceType = this.ProgramKey.PackType;
					e2.ErrorType = ErrorsType.INFORMATION;
					e2.Time = DateTime.Now;
					e2.Key = this.ProgramKey.Program;
					e2.Description = Application.Current.FindResource("Message_CantGetFunction") as string;
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e2);
					return;
				}
				FunctionSelectedModel functionSelectedModel = new FunctionSelectedModel();
				functionSelectedModel.ProgramKey = new PackageKey(codeSpecificationMainWindow.ProgramKey.Program, TzpType.Code);
				functionSelectedModel.Target = fglComponent.FunctionName;
				EventAggregatorManager.Global.GetEvent<FunctionSelectedEvent>().Publish(functionSelectedModel);
			}
		}

		// Token: 0x06000128 RID: 296 RVA: 0x0000B8CC File Offset: 0x00009ACC
		private void CanDelete(object target, CanExecuteRoutedEventArgs args)
		{
			IEditableCollectionView items = this.componentList.Items;
			if (items.IsAddingNew || items.IsEditingItem)
			{
				args.CanExecute = false;
				return;
			}
			args.CanExecute = true;
		}

		// Token: 0x06000129 RID: 297 RVA: 0x0000B904 File Offset: 0x00009B04
		private void ExecuteDelete(object target, ExecutedRoutedEventArgs args)
		{
			FglComponent fglComponent = this.componentList.SelectedItem as FglComponent;
			object parameter = args.Parameter;
			if (parameter is FglComponent)
			{
				this.SpecificationInfo.Remove(parameter as FglComponent);
				return;
			}
			if (parameter is FglTest)
			{
				fglComponent.Remove(parameter as FglTest);
				return;
			}
			if (parameter is FglDimension)
			{
				fglComponent.Remove(parameter as FglDimension);
				return;
			}
			if (parameter is FglFrontComp)
			{
				fglComponent.Remove(parameter as FglFrontComp);
				return;
			}
			if (parameter is FglKeyword)
			{
				fglComponent.Remove(parameter as FglKeyword);
				return;
			}
			if (parameter is FglParameter)
			{
				fglComponent.Remove(parameter as FglParameter);
			}
		}

		// Token: 0x0600012A RID: 298 RVA: 0x0000B9AC File Offset: 0x00009BAC
		private void CanAdd(object target, CanExecuteRoutedEventArgs args)
		{
			FglComponent fglComponent = this.componentList.SelectedItem as FglComponent;
			string text = args.Parameter as string;
			if (fglComponent == null && typeof(FglComponent).Name != text)
			{
				args.CanExecute = false;
				return;
			}
			args.CanExecute = true;
		}

		// Token: 0x0600012B RID: 299 RVA: 0x0000BA00 File Offset: 0x00009C00
		private void ExecuteAdd(object target, ExecutedRoutedEventArgs args)
		{
			FglComponent fglComponent = this.componentList.SelectedItem as FglComponent;
			string text = args.Parameter as string;
			if (text == typeof(FglComponent).Name)
			{
				FglComponent fglComponent2 = null;
				try
				{
					fglComponent2 = FglComponent.Initial(this.SpecificationInfo);
				}
				catch (Exception ex)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex.Message), "Error", MessageBoxButton.OK);
					fglComponent2 = null;
				}
				if (fglComponent2 != null)
				{
					this.SpecificationInfo.Add(fglComponent2);
					return;
				}
			}
			else if (text == typeof(FglTest).Name)
			{
				FglTest fglTest = null;
				try
				{
					fglTest = fglComponent.InitialTest();
				}
				catch (Exception ex2)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex2.Message), "Error", MessageBoxButton.OK);
					fglTest = null;
				}
				if (fglTest != null)
				{
					fglComponent.Add(fglTest);
					return;
				}
			}
			else if (text == typeof(FglFrontComp).Name)
			{
				FglFrontComp fglFrontComp = null;
				try
				{
					fglFrontComp = fglComponent.InitialFrontComp();
				}
				catch (Exception ex3)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex3.Message), "Error", MessageBoxButton.OK);
					fglFrontComp = null;
				}
				if (fglFrontComp != null)
				{
					fglComponent.Add(fglFrontComp);
					return;
				}
			}
			else if (text == typeof(FglKeyword).Name)
			{
				FglKeyword fglKeyword = null;
				try
				{
					fglKeyword = fglComponent.InitialKeyword();
				}
				catch (Exception ex4)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex4.Message), "Error", MessageBoxButton.OK);
					fglKeyword = null;
				}
				if (fglKeyword != null)
				{
					fglComponent.Add(fglKeyword);
					return;
				}
			}
			else if (text == "input")
			{
				FglParameter fglParameter = null;
				try
				{
					fglParameter = fglComponent.InitialParameter(false);
				}
				catch (Exception ex5)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex5.Message), "Error", MessageBoxButton.OK);
					fglParameter = null;
				}
				if (fglParameter != null)
				{
					fglComponent.Add(fglParameter);
					return;
				}
			}
			else if (text == "return")
			{
				FglParameter fglParameter2 = null;
				try
				{
					fglParameter2 = fglComponent.InitialParameter(true);
				}
				catch (Exception ex6)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex6.Message), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK);
					fglParameter2 = null;
				}
				if (fglParameter2 != null)
				{
					fglComponent.Add(fglParameter2);
					return;
				}
			}
			else if (text == typeof(FglDimension).Name)
			{
				FglDimension fglDimension;
				try
				{
					fglDimension = fglComponent.InitialDimension();
				}
				catch (Exception ex7)
				{
					DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_CreateFailed") as string, ex7.Message), "Error", MessageBoxButton.OK);
					fglDimension = null;
				}
				if (fglDimension != null)
				{
					fglComponent.Add(fglDimension);
				}
			}
		}

		// Token: 0x0600012C RID: 300 RVA: 0x0000BD4C File Offset: 0x00009F4C
		private void CanCreateFunction(object target, CanExecuteRoutedEventArgs e)
		{
			if (this.componentList.Items.Count > 0)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600012D RID: 301 RVA: 0x0000BD70 File Offset: 0x00009F70
		private void ExecuteCreateFunction(object target, ExecutedRoutedEventArgs e)
		{
			foreach (string text in FunctionGenerator.Generate(this.SpecificationInfo))
			{
				try
				{
					CreateFunctionContentArgs createFunctionContentArgs = new CreateFunctionContentArgs();
					createFunctionContentArgs.Content = text;
					createFunctionContentArgs.ProgramKey = this.SpecificationInfo.ProgramKey;
					EventController.GetInstance().GetEvent<CreateFunctionEvent>().Publish(createFunctionContentArgs);
				}
				catch (Exception ex)
				{
					DocumentErrorsEventArgs e2 = new DocumentErrorsEventArgs();
					e2.ProgramKey = this.SpecificationInfo.ProgramKey;
					e2.SourceType = this.SpecificationInfo.ProgramKey.PackType;
					e2.ErrorType = ErrorsType.ERROR;
					e2.Time = DateTime.Now;
					e2.Description = string.Format("{0}, {1}", ex.Message, Application.Current.FindResource("Message_NotProduce") as string);
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e2);
				}
			}
		}

		// Token: 0x0600012E RID: 302 RVA: 0x0000BE80 File Offset: 0x0000A080
		private void OnTzpFileClose(PackageKey key)
		{
			if (key == this.ProgramKey)
			{
				this.Dispose();
			}
		}

		// Token: 0x0600012F RID: 303 RVA: 0x0000BE96 File Offset: 0x0000A096
		private void OnSaveSetting(PackageKey key)
		{
			if (key == this.ProgramKey)
			{
				SettingManager.Get().GetTzpManger(this.ProgramKey).SaveSpecificationForCode();
			}
		}

		// Token: 0x06000130 RID: 304 RVA: 0x0000BEBC File Offset: 0x0000A0BC
		public void Dispose()
		{
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Unsubscribe(new Action<PackageKey>(this.OnSaveSetting));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.OnTzpFileClose));
			base.CommandBindings.Clear();
			this.componentList.SelectionChanged -= this.componentList_SelectionChanged;
			base.Resources.Remove("ForDBs");
			base.Resources.Remove("Dimensions");
			base.Resources.Remove("DBs");
		}

		// Token: 0x17000022 RID: 34
		// (get) Token: 0x06000131 RID: 305 RVA: 0x0000BF51 File Offset: 0x0000A151
		public FrameworkElement FormSpecEditorContent
		{
			get
			{
				return this._specBox;
			}
		}

		// Token: 0x0400008B RID: 139
		private FglSpecification SpecificationInfo;

		// Token: 0x0400008C RID: 140
		private TextBox _specBox = new TextBox
		{
			AcceptsReturn = true,
			HorizontalScrollBarVisibility = ScrollBarVisibility.Auto,
			VerticalScrollBarVisibility = ScrollBarVisibility.Auto
		};
	}
}
