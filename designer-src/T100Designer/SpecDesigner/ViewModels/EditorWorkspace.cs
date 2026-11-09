using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Helper;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000003 RID: 3
	public class EditorWorkspace : INotifyPropertyChanged
	{
		// Token: 0x06000007 RID: 7 RVA: 0x00002144 File Offset: 0x00000344
		protected EditorWorkspace()
		{
			EventAggregatorManager.Global.GetEvent<TzpFileLoaded>().Subscribe(delegate(PackageKey key)
			{
				EventAggregatorManager.CreateInstance(key);
				this.CurrentOpening = key;
				this.ActiveDocument = this.Open(key);
				this.CurrentOpening = null;
			});
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Subscribe(delegate(SearchResultInfo args)
			{
				if (this.ActiveDocument.Key != args.ProgramKey)
				{
					this.ActiveDocument = this._editors.Where<FileViewModel>((FileViewModel editor) => editor.Key.Equals(args.ProgramKey)).ElementAtOrDefault<FileViewModel>(0);
				}
			});
			EventAggregatorManager.Global.GetEvent<FunctionSelectedEvent>().Subscribe(new Action<FunctionSelectedModel>(this.OnFunctionSelected));
		}

		// Token: 0x06000008 RID: 8 RVA: 0x000021C4 File Offset: 0x000003C4
		private void OnFunctionSelected(FunctionSelectedModel model)
		{
			if (model.ProgramKey != this.ActiveDocument.Key)
			{
				foreach (FileViewModel fileViewModel in this._editors)
				{
					if (model.ProgramKey == fileViewModel.Key)
					{
						this.ActiveDocument = fileViewModel;
					}
				}
			}
			if (model.ProgramKey == this.ActiveDocument.Key)
			{
				EventAggregatorManager.Get(model.ProgramKey).GetEvent<FunctionSelectedEvent>().Publish(model);
			}
		}

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000009 RID: 9 RVA: 0x0000226C File Offset: 0x0000046C
		// (set) Token: 0x0600000A RID: 10 RVA: 0x00002274 File Offset: 0x00000474
		internal PackageKey CurrentOpening { get; set; }

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600000B RID: 11 RVA: 0x0000227D File Offset: 0x0000047D
		public PackageKey CurrentProgram
		{
			get
			{
				if (this.CurrentOpening != null)
				{
					return this.CurrentOpening;
				}
				if (this.ActiveDocument != null)
				{
					return this.ActiveDocument.Key;
				}
				return null;
			}
		}

		// Token: 0x0600000C RID: 12 RVA: 0x000022AC File Offset: 0x000004AC
		public bool CheckUnsavedFile()
		{
			FileViewModel fileViewModel;
			do
			{
				fileViewModel = this._editors.ElementAtOrDefault<FileViewModel>(0);
				if (fileViewModel == null)
				{
					return true;
				}
			}
			while (this.Close(fileViewModel));
			return false;
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x0600000D RID: 13 RVA: 0x000022D8 File Offset: 0x000004D8
		// (remove) Token: 0x0600000E RID: 14 RVA: 0x00002310 File Offset: 0x00000510
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600000F RID: 15 RVA: 0x00002345 File Offset: 0x00000545
		public void RaisePropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000010 RID: 16 RVA: 0x00002361 File Offset: 0x00000561
		public static EditorWorkspace This
		{
			get
			{
				return EditorWorkspace._this;
			}
		}

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x06000011 RID: 17 RVA: 0x00002368 File Offset: 0x00000568
		// (set) Token: 0x06000012 RID: 18 RVA: 0x00002370 File Offset: 0x00000570
		public FileViewModel ActiveDocument
		{
			get
			{
				return this._activeDocument;
			}
			set
			{
				PackageKey packageKey = ((this._activeDocument == null) ? null : this._activeDocument.Key);
				if (this._activeDocument != value)
				{
					this._activeDocument = value;
					if (value != null && !this._activeDocument.IsSelected)
					{
						this._activeDocument.IsSelected = true;
					}
				}
				PackageKey packageKey2 = ((this._activeDocument == null) ? null : this._activeDocument.Key);
				Application.Current.MainWindow.Tag = packageKey2;
				if (packageKey != packageKey2)
				{
					ProgramSelectionChangedEventArgs e = new ProgramSelectionChangedEventArgs
					{
						OldProgramKey = packageKey,
						NewProgramKey = packageKey2
					};
					EventAggregatorManager.Global.GetEvent<ProgramSelectionChangedEvent>().Publish(e);
				}
				this.RaisePropertyChanged("ActiveDocument");
				this.GetSpecInfoFlag(packageKey2);
				if (this.ActiveDocumentChanged != null)
				{
					this.ActiveDocumentChanged(this, EventArgs.Empty);
				}
			}
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x06000013 RID: 19 RVA: 0x00002444 File Offset: 0x00000644
		// (remove) Token: 0x06000014 RID: 20 RVA: 0x0000247C File Offset: 0x0000067C
		public event EventHandler ActiveDocumentChanged;

		// Token: 0x06000015 RID: 21 RVA: 0x000024B4 File Offset: 0x000006B4
		private void GetSpecInfoFlag(PackageKey newProgramName)
		{
			TzpType tzpType = ((null == newProgramName) ? TzpType.None : newProgramName.PackType);
			if (TzpManager.Current.Type == TzpType.Form && tzpType == TzpType.Form)
			{
				this.IsFreeStyle = TzpManager.Current.SpecificationInfo.IsFreeStyle;
				return;
			}
			this.IsFreeStyle = false;
			this.IsSectionModify = false;
		}

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x06000016 RID: 22 RVA: 0x00002509 File Offset: 0x00000709
		// (set) Token: 0x06000017 RID: 23 RVA: 0x00002510 File Offset: 0x00000710
		public bool IsFreeStyle
		{
			get
			{
				return EditorWorkspace._isFreeStyle;
			}
			set
			{
				if (EditorWorkspace._isFreeStyle != value)
				{
					EditorWorkspace._isFreeStyle = value;
				}
				this.RaisePropertyChanged("IsFreeStyle");
			}
		}

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000018 RID: 24 RVA: 0x0000252B File Offset: 0x0000072B
		// (set) Token: 0x06000019 RID: 25 RVA: 0x00002532 File Offset: 0x00000732
		public bool IsSectionModify
		{
			get
			{
				return EditorWorkspace._isSectionModify;
			}
			set
			{
				if (EditorWorkspace._isSectionModify != value)
				{
					EditorWorkspace._isSectionModify = value;
				}
				this.RaisePropertyChanged("IsSectionModify");
			}
		}

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x0600001A RID: 26 RVA: 0x0000254D File Offset: 0x0000074D
		public ReadOnlyObservableCollection<FileViewModel> Editors
		{
			get
			{
				if (this._readonyFiles == null)
				{
					this._readonyFiles = new ReadOnlyObservableCollection<FileViewModel>(this._editors);
				}
				return this._readonyFiles;
			}
		}

		// Token: 0x0600001B RID: 27 RVA: 0x0000258C File Offset: 0x0000078C
		public FileViewModel Open(PackageKey key)
		{
			FileViewModel fileViewModel = this._editors.FirstOrDefault<FileViewModel>((FileViewModel fm) => fm.Key == key);
			if (fileViewModel != null)
			{
				return fileViewModel;
			}
			fileViewModel = FileViewModelFactory.CreateWithKey(key);
			this._editors.Add(fileViewModel);
			return fileViewModel;
		}

		// Token: 0x0600001C RID: 28 RVA: 0x000025DC File Offset: 0x000007DC
		public void CloseAll()
		{
			int num = 0;
			while (this.Editors.Count > 0 && num < this.Editors.Count)
			{
				this.Editors[num].CloseCommand.Execute(null);
			}
		}

		// Token: 0x0600001D RID: 29 RVA: 0x0000263C File Offset: 0x0000083C
		public bool Close(FileViewModel editor)
		{
			if (editor == null)
			{
				return true;
			}
			if (editor.IsModified)
			{
				string text = Application.Current.FindResource("Message_SaveFile") as string;
				string text2 = string.Format(text, editor.Key.Program);
				string text3 = Application.Current.FindResource("Message_SaveBeforeClose") as string;
				MessageBoxResult messageBoxResult = ((!PreferenceManager.Current.Settings.RemaindSave) ? MessageBoxResult.Yes : DesignerMessageBox.Show(text2, text3, MessageBoxButton.YesNoCancel, MessageBoxImage.Question));
				MessageBoxResult messageBoxResult2 = messageBoxResult;
				if (messageBoxResult2 == MessageBoxResult.Cancel)
				{
					return false;
				}
				if (messageBoxResult2 == MessageBoxResult.Yes)
				{
					SettingManager.Get().GetTzpManger(editor.Key).TzpSaved += delegate(object s, TzpSavedEventArgs e)
					{
						this.Remove(editor);
					};
					SettingManager.Get().SaveSetting(editor.Key);
				}
				else
				{
					this.Remove(editor);
				}
			}
			else
			{
				this.Remove(editor);
			}
			return true;
		}

		// Token: 0x0600001E RID: 30 RVA: 0x00002754 File Offset: 0x00000954
		private void Remove(FileViewModel editor)
		{
			if (this.ActiveDocument == editor)
			{
				this.ActiveDocument = null;
				this.IsSectionModify = false;
				this.IsFreeStyle = false;
			}
			this._editors.Remove(editor);
		}

		// Token: 0x0600001F RID: 31 RVA: 0x00002781 File Offset: 0x00000981
		public void CloseOpening()
		{
			if (this.CurrentOpening != null)
			{
				EventAggregatorManager.Global.GetEvent<TzpFileClose>().Publish(this.CurrentOpening);
				EventAggregatorManager.Remove(this.CurrentOpening);
				this.RemoveDocumentFromKey(this.CurrentOpening);
			}
		}

		// Token: 0x06000020 RID: 32 RVA: 0x000027D8 File Offset: 0x000009D8
		public void SetActiveDocumentFromKey(PackageKey key)
		{
			FileViewModel fileViewModel = this._editors.Where<FileViewModel>((FileViewModel m) => m.Key == key).ElementAtOrDefault<FileViewModel>(0);
			if (fileViewModel != null)
			{
				this.ActiveDocument = fileViewModel;
			}
		}

		// Token: 0x06000021 RID: 33 RVA: 0x00002838 File Offset: 0x00000A38
		private void RemoveDocumentFromKey(PackageKey key)
		{
			FileViewModel fileViewModel = this._editors.Where<FileViewModel>((FileViewModel m) => m.Key == key).ElementAtOrDefault<FileViewModel>(0);
			if (fileViewModel != null)
			{
				fileViewModel.CloseCommand.Execute(null);
				this.Remove(fileViewModel);
			}
		}

		// Token: 0x06000022 RID: 34 RVA: 0x000028C0 File Offset: 0x00000AC0
		internal bool IsExists(string program, TzpType type)
		{
			return this.Editors.Where<FileViewModel>((FileViewModel f) => f.Key.Program == program && f.Key.PackType == type).Count<FileViewModel>() > 0;
		}

		// Token: 0x06000023 RID: 35 RVA: 0x0000291C File Offset: 0x00000B1C
		internal FileViewModel FindFileViewModel(PackageKey packageKey)
		{
			IEnumerable<FileViewModel> enumerable = this._editors.Where<FileViewModel>((FileViewModel f) => f.Key == packageKey);
			if (1 == enumerable.Count<FileViewModel>())
			{
				return enumerable.ElementAtOrDefault<FileViewModel>(0);
			}
			return null;
		}

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000024 RID: 36 RVA: 0x00002960 File Offset: 0x00000B60
		// (set) Token: 0x06000025 RID: 37 RVA: 0x00002968 File Offset: 0x00000B68
		public bool SpecPropertyEditorLayout_IsChecked
		{
			get
			{
				return this._SpecPropertyEditorLayout_IsChecked;
			}
			set
			{
				this._SpecPropertyEditorLayout_IsChecked = value;
				this.RaisePropertyChanged("SpecPropertyEditorLayout_IsChecked");
			}
		}

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x06000026 RID: 38 RVA: 0x0000297C File Offset: 0x00000B7C
		// (set) Token: 0x06000027 RID: 39 RVA: 0x00002984 File Offset: 0x00000B84
		public bool FormStructureLayout_IsChecked
		{
			get
			{
				return this._FormStructureLayout_IsChecked;
			}
			set
			{
				this._FormStructureLayout_IsChecked = value;
				this.RaisePropertyChanged("FormStructureLayout_IsChecked");
			}
		}

		// Token: 0x1700000C RID: 12
		// (get) Token: 0x06000028 RID: 40 RVA: 0x00002998 File Offset: 0x00000B98
		// (set) Token: 0x06000029 RID: 41 RVA: 0x000029A0 File Offset: 0x00000BA0
		public bool FunctionListLayout_IsChecked
		{
			get
			{
				return this._FunctionListLayout_IsChecked;
			}
			set
			{
				this._FunctionListLayout_IsChecked = value;
				this.RaisePropertyChanged("FunctionListLayout_IsChecked");
			}
		}

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x0600002A RID: 42 RVA: 0x000029B4 File Offset: 0x00000BB4
		// (set) Token: 0x0600002B RID: 43 RVA: 0x000029BC File Offset: 0x00000BBC
		public bool SpecEditorLayout_IsChecked
		{
			get
			{
				return this._SpecEditorLayout_IsChecked;
			}
			set
			{
				this._SpecEditorLayout_IsChecked = value;
				this.RaisePropertyChanged("SpecEditorLayout_IsChecked");
			}
		}

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x0600002C RID: 44 RVA: 0x000029D0 File Offset: 0x00000BD0
		// (set) Token: 0x0600002D RID: 45 RVA: 0x000029D8 File Offset: 0x00000BD8
		public bool SpecErrorsLayout_IsChecked
		{
			get
			{
				return this._SpecErrorsLayout_IsChecked;
			}
			set
			{
				this._SpecErrorsLayout_IsChecked = value;
				this.RaisePropertyChanged("SpecErrorsLayout_IsChecked");
			}
		}

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x0600002E RID: 46 RVA: 0x000029EC File Offset: 0x00000BEC
		// (set) Token: 0x0600002F RID: 47 RVA: 0x000029F4 File Offset: 0x00000BF4
		public bool BookmarksLayout_IsChecked
		{
			get
			{
				return this._BookmarksLayout_IsChecked;
			}
			set
			{
				this._BookmarksLayout_IsChecked = value;
				this.RaisePropertyChanged("BookmarksLayout_IsChecked");
			}
		}

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000030 RID: 48 RVA: 0x00002A08 File Offset: 0x00000C08
		// (set) Token: 0x06000031 RID: 49 RVA: 0x00002A10 File Offset: 0x00000C10
		public bool WelcomeLayout_IsChecked
		{
			get
			{
				return this._WelcomeLayout_IsChecked;
			}
			set
			{
				this._WelcomeLayout_IsChecked = value;
				this.RaisePropertyChanged("WelcomeLayout_IsChecked");
			}
		}

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000032 RID: 50 RVA: 0x00002A24 File Offset: 0x00000C24
		// (set) Token: 0x06000033 RID: 51 RVA: 0x00002A2C File Offset: 0x00000C2C
		public bool DatabaseSourceLayout_IsChecked
		{
			get
			{
				return this._DatabaseSourceLayout_IsChecked;
			}
			set
			{
				this._DatabaseSourceLayout_IsChecked = value;
				this.RaisePropertyChanged("DatabaseSourceLayout_IsChecked");
			}
		}

		// Token: 0x17000012 RID: 18
		// (get) Token: 0x06000034 RID: 52 RVA: 0x00002A40 File Offset: 0x00000C40
		// (set) Token: 0x06000035 RID: 53 RVA: 0x00002A54 File Offset: 0x00000C54
		public bool SimpleDownloadDialog
		{
			get
			{
				return !PreferenceManager.Current.Settings.StandardView;
			}
			set
			{
				PreferenceManager.Current.Settings.StandardView = !value;
			}
		}

		// Token: 0x04000004 RID: 4
		private static EditorWorkspace _this = new EditorWorkspace();

		// Token: 0x04000005 RID: 5
		private FileViewModel _activeDocument;

		// Token: 0x04000007 RID: 7
		public static bool _isFreeStyle;

		// Token: 0x04000008 RID: 8
		public static bool _isSectionModify;

		// Token: 0x04000009 RID: 9
		private ObservableCollection<FileViewModel> _editors = new ObservableCollection<FileViewModel>();

		// Token: 0x0400000A RID: 10
		private ReadOnlyObservableCollection<FileViewModel> _readonyFiles;

		// Token: 0x0400000B RID: 11
		private bool _SpecPropertyEditorLayout_IsChecked;

		// Token: 0x0400000C RID: 12
		private bool _FormStructureLayout_IsChecked;

		// Token: 0x0400000D RID: 13
		private bool _FunctionListLayout_IsChecked;

		// Token: 0x0400000E RID: 14
		private bool _SpecEditorLayout_IsChecked;

		// Token: 0x0400000F RID: 15
		private bool _SpecErrorsLayout_IsChecked;

		// Token: 0x04000010 RID: 16
		private bool _BookmarksLayout_IsChecked;

		// Token: 0x04000011 RID: 17
		private bool _WelcomeLayout_IsChecked;

		// Token: 0x04000012 RID: 18
		private bool _DatabaseSourceLayout_IsChecked;
	}
}
