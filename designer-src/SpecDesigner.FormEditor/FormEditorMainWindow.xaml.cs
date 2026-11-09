using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using Microsoft.Practices.Prism.Events;
using SpecDesigner.FormEditor.Views;
using SpecDesigner.SpecEditor;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor
{
	// Token: 0x02000052 RID: 82
	public partial class FormEditorMainWindow : UserControl, IAvalonFormPropertiesLayout, IAvalonFormSpecEditorLayout, IAvalonFormStructureLayout
	{
		// Token: 0x17000096 RID: 150
		// (get) Token: 0x06000349 RID: 841 RVA: 0x00012309 File Offset: 0x00010509
		// (set) Token: 0x0600034A RID: 842 RVA: 0x00012311 File Offset: 0x00010511
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x0600034B RID: 843 RVA: 0x00012340 File Offset: 0x00010540
		public FormEditorMainWindow(PackageKey key)
		{
			this.InitializeComponent();
			this.ProgramKey = key;
			Cursor wait = Cursors.Wait;
			base.Cursor = wait;
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Subscribe(new Action<PackageKey>(this.GeneroFormFileSubscribe));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.Subscribe_TzpFileClose));
			base.Loaded += this.FormEditorMainWindow_Loaded;
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsSimpleForm)
			{
				this.workspace.DesignContainer.ColumnDefinitions[1].Width = new GridLength(0.0);
				this.workspace.actionDefaults.Visibility = Visibility.Collapsed;
				this.workspace.toolbar.Visibility = Visibility.Collapsed;
				this.workspace.widgetBox.Visibility = Visibility.Collapsed;
				this.workspace.simpleFormWidgetBox.Visibility = Visibility.Visible;
			}
			else
			{
				this.workspace.widgetBox.Visibility = Visibility.Visible;
				this.workspace.simpleFormWidgetBox.Visibility = Visibility.Collapsed;
			}
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Subscribe(new Action<PackageKey>(((SpecPropertyEditor)this.FormPropertiesContent).RenderPropertyGrid));
			EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Subscribe(new Action<SpecArgs>(((SpecPropertyEditor)this.FormPropertiesContent).LoadSpecification), ThreadOption.BackgroundThread);
			EventAggregatorManager.Global.GetEvent<MultiComponentSelectedEvent>().Subscribe(new Action<MultiSelectionArgs>(((SpecPropertyEditor)this.FormPropertiesContent).LoadSpecificationWithComponents), ThreadOption.BackgroundThread);
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(((SpecPropertyEditor)this.FormPropertiesContent).SubscribeTzpFileClosed));
			EventAggregatorManager.Global.GetEvent<ProgramSelectionChangedEvent>().Subscribe(new Action<ProgramSelectionChangedEventArgs>(((SpecPropertyEditor)this.FormPropertiesContent).ProgramSelectionChanged));
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Subscribe(new Action<string>(((SpecPropertyEditor)this.FormPropertiesContent).RenderPropertyGrid));
			EventAggregatorManager.Global.GetEvent<HideSpecPropertiesEvent>().Subscribe(delegate(SpecArgs s)
			{
				((SpecPropertyEditor)this.FormPropertiesContent).Init();
			});
			EventAggregatorManager.Global.GetEvent<TzpFileLoaded>().Subscribe(delegate(PackageKey s)
			{
				((SpecPropertyEditor)this.FormPropertiesContent).Init();
			});
		}

		// Token: 0x0600034C RID: 844 RVA: 0x0001258C File Offset: 0x0001078C
		public void Subscribe_TzpFileClose(PackageKey key)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.Subscribe_TzpFileClose));
			ScreenRecordViewer.This.ProgramKey = null;
			base.DataContext = Binding.DoNothing;
			if (this.workspace != null)
			{
				this.workspace.Dispose();
			}
			base.CommandBindings.Clear();
			TabItem tabItem = base.Parent as TabItem;
			if (tabItem != null)
			{
				tabItem.Content = null;
			}
		}

		// Token: 0x0600034D RID: 845 RVA: 0x00012610 File Offset: 0x00010810
		private void FormEditorMainWindow_Loaded(object sender, RoutedEventArgs e)
		{
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Undo, new ExecutedRoutedEventHandler(this.OnUndo), new CanExecuteRoutedEventHandler(this.CanUndo)));
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Redo, new ExecutedRoutedEventHandler(this.OnRedo), new CanExecuteRoutedEventHandler(this.CanRedo)));
			base.Cursor = null;
			base.Loaded -= this.FormEditorMainWindow_Loaded;
		}

		// Token: 0x0600034E RID: 846 RVA: 0x00012692 File Offset: 0x00010892
		private void GeneroFormFileSubscribe(PackageKey key)
		{
			this.ProgramKey = key;
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Unsubscribe(new Action<PackageKey>(this.GeneroFormFileSubscribe));
			base.DataContext = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
		}

		// Token: 0x17000097 RID: 151
		// (get) Token: 0x0600034F RID: 847 RVA: 0x000126D1 File Offset: 0x000108D1
		public string ProgramName
		{
			get
			{
				return this.ProgramKey.Program;
			}
		}

		// Token: 0x06000350 RID: 848 RVA: 0x000126DE File Offset: 0x000108DE
		private void CanUndo(object target, CanExecuteRoutedEventArgs args)
		{
			if (SettingManager.Get().GetUndoRedoManager(this.ProgramKey).UndoCount > 0)
			{
				args.CanExecute = true;
				return;
			}
			args.CanExecute = false;
		}

		// Token: 0x06000351 RID: 849 RVA: 0x00012707 File Offset: 0x00010907
		private void OnUndo(object target, ExecutedRoutedEventArgs args)
		{
			SettingManager.Get().GetUndoRedoManager(this.ProgramKey).Undo();
		}

		// Token: 0x06000352 RID: 850 RVA: 0x0001271E File Offset: 0x0001091E
		private void CanRedo(object target, CanExecuteRoutedEventArgs args)
		{
			if (SettingManager.Get().GetUndoRedoManager(this.ProgramKey).RedoCount > 0)
			{
				args.CanExecute = true;
				return;
			}
			args.CanExecute = false;
		}

		// Token: 0x06000353 RID: 851 RVA: 0x00012747 File Offset: 0x00010947
		private void OnRedo(object target, ExecutedRoutedEventArgs args)
		{
			SettingManager.Get().GetUndoRedoManager(this.ProgramKey).Redo();
		}

		// Token: 0x17000098 RID: 152
		// (get) Token: 0x06000354 RID: 852 RVA: 0x0001275E File Offset: 0x0001095E
		public FrameworkElement FormPropertiesContent
		{
			get
			{
				return SpecPropertyEditor.This;
			}
		}

		// Token: 0x17000099 RID: 153
		// (get) Token: 0x06000355 RID: 853 RVA: 0x00012765 File Offset: 0x00010965
		public FrameworkElement FormSpecEditorContent
		{
			get
			{
				if (this._specEditor == null)
				{
					this._specEditor = new SpecEditor(this.ProgramKey);
				}
				return this._specEditor;
			}
		}

		// Token: 0x1700009A RID: 154
		// (get) Token: 0x06000356 RID: 854 RVA: 0x00012786 File Offset: 0x00010986
		public FrameworkElement FormStructureContent
		{
			get
			{
				ScreenRecordViewer.This.ProgramKey = this.ProgramKey;
				return ScreenRecordViewer.This;
			}
		}

		// Token: 0x1700009B RID: 155
		// (get) Token: 0x06000357 RID: 855 RVA: 0x0001279D File Offset: 0x0001099D
		public FrameworkElement FormDatabaseSource
		{
			get
			{
				DatabaseSourceViewModel.This.ProgramKey = this.ProgramKey;
				if (this.dsv == null)
				{
					this.dsv = new DatabaseSourceViewer();
				}
				return this.dsv;
			}
		}

		// Token: 0x040001AD RID: 429
		private SpecEditor _specEditor;

		// Token: 0x040001AE RID: 430
		private DatabaseSourceViewer dsv;
	}
}
