using System;
using System.Windows;
using System.Windows.Input;
using System.Windows.Media;
using SpecDesignerCommon;
using SpecDesignerCommon.Bookmark;
using SpecDesignerCommon.Events;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000013 RID: 19
	public abstract class FileViewModel : PaneViewModel
	{
		// Token: 0x060001C3 RID: 451 RVA: 0x00007F1C File Offset: 0x0000611C
		protected FileViewModel()
		{
			this.IsModified = false;
			this._closeCommand = new RelayCommand(delegate(object p)
			{
				this.OnClose();
			}, (object p) => this.CanClose());
		}

		// Token: 0x060001C4 RID: 452 RVA: 0x00007F74 File Offset: 0x00006174
		public FileViewModel(PackageKey key)
			: this()
		{
			this.Key = key;
			SettingManager.Get().GetTzpManger(this.Key).TzpSaved += new TzpManager.TzpSavedHandler(this.OnTzpSaved);
			this.FilePath = SettingManager.Get().GetTzpManger(this.Key).ZipFile;
		}

		// Token: 0x060001C5 RID: 453 RVA: 0x00007FCA File Offset: 0x000061CA
		private void OnTzpSaved(object sender, EventArgs args)
		{
			this.IsModified = false;
		}

		// Token: 0x17000080 RID: 128
		// (get) Token: 0x060001C6 RID: 454 RVA: 0x00007FD3 File Offset: 0x000061D3
		// (set) Token: 0x060001C7 RID: 455 RVA: 0x00007FDB File Offset: 0x000061DB
		public PackageKey Key { get; private set; }

		// Token: 0x17000081 RID: 129
		// (get) Token: 0x060001C8 RID: 456 RVA: 0x00007FE4 File Offset: 0x000061E4
		public new string Title
		{
			get
			{
				return this.Key.Program + (this.IsModified ? "*" : "");
			}
		}

		// Token: 0x17000082 RID: 130
		// (get) Token: 0x060001C9 RID: 457 RVA: 0x0000800A File Offset: 0x0000620A
		// (set) Token: 0x060001CA RID: 458 RVA: 0x00008012 File Offset: 0x00006212
		public string FilePath
		{
			get
			{
				return this._filePath;
			}
			private set
			{
				this._filePath = value;
				base.RaisePropertyChanged("FilePath");
			}
		}

		// Token: 0x17000083 RID: 131
		// (get) Token: 0x060001CB RID: 459 RVA: 0x00008026 File Offset: 0x00006226
		public ICommand CloseCommand
		{
			get
			{
				return this._closeCommand;
			}
		}

		// Token: 0x060001CC RID: 460 RVA: 0x0000802E File Offset: 0x0000622E
		public virtual bool CanClose()
		{
			return true;
		}

		// Token: 0x060001CD RID: 461 RVA: 0x00008034 File Offset: 0x00006234
		public virtual void OnClose()
		{
			SettingManager.Get().GetTzpManger(this.Key).TzpSaved -= new TzpManager.TzpSavedHandler(this.OnTzpSaved);
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Publish(this.Key);
			EventAggregatorManager.Remove(this.Key);
		}

		// Token: 0x17000084 RID: 132
		// (get) Token: 0x060001CE RID: 462 RVA: 0x00008082 File Offset: 0x00006282
		// (set) Token: 0x060001CF RID: 463 RVA: 0x0000808A File Offset: 0x0000628A
		public bool IsModified
		{
			get
			{
				return this._isModified;
			}
			set
			{
				if (this._isModified != value)
				{
					this._isModified = value;
					base.RaisePropertyChanged("IsModified");
					base.RaisePropertyChanged("Title");
				}
			}
		}

		// Token: 0x17000085 RID: 133
		// (get) Token: 0x060001D0 RID: 464 RVA: 0x000080B2 File Offset: 0x000062B2
		// (set) Token: 0x060001D1 RID: 465 RVA: 0x000080BC File Offset: 0x000062BC
		public FrameworkElement UI
		{
			get
			{
				return this._ui;
			}
			set
			{
				if (this._ui != null)
				{
					return;
				}
				this._ui = value;
				if (value != null)
				{
					base.ContentId = string.Format("{0}_{1}", this.Title, this._ui.GetType().Name);
				}
				base.RaisePropertyChanged("UI");
			}
		}

		// Token: 0x17000086 RID: 134
		// (get) Token: 0x060001D2 RID: 466 RVA: 0x0000810D File Offset: 0x0000630D
		public virtual FrameworkElement PropertiesUI
		{
			get
			{
				return null;
			}
		}

		// Token: 0x17000087 RID: 135
		// (get) Token: 0x060001D3 RID: 467 RVA: 0x00008110 File Offset: 0x00006310
		public virtual FrameworkElement Structure
		{
			get
			{
				return null;
			}
		}

		// Token: 0x17000088 RID: 136
		// (get) Token: 0x060001D4 RID: 468 RVA: 0x00008113 File Offset: 0x00006313
		public virtual FrameworkElement SpecUI
		{
			get
			{
				return null;
			}
		}

		// Token: 0x17000089 RID: 137
		// (get) Token: 0x060001D5 RID: 469 RVA: 0x00008116 File Offset: 0x00006316
		public virtual IBookmarkMargin BookmarkManager
		{
			get
			{
				return null;
			}
		}

		// Token: 0x1700008A RID: 138
		// (get) Token: 0x060001D6 RID: 470 RVA: 0x00008119 File Offset: 0x00006319
		// (set) Token: 0x060001D7 RID: 471 RVA: 0x00008121 File Offset: 0x00006321
		public ImageSource IconSource
		{
			get
			{
				return this._iconSource;
			}
			set
			{
				this._iconSource = value;
				base.RaisePropertyChanged("IconSource");
			}
		}

		// Token: 0x1700008B RID: 139
		// (get) Token: 0x060001D8 RID: 472 RVA: 0x00008135 File Offset: 0x00006335
		public virtual FrameworkElement DatabaseSource
		{
			get
			{
				return null;
			}
		}

		// Token: 0x1700008C RID: 140
		// (get) Token: 0x060001D9 RID: 473 RVA: 0x00008138 File Offset: 0x00006338
		public virtual string SelectionContent
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x040000AF RID: 175
		private string _filePath = string.Empty;

		// Token: 0x040000B0 RID: 176
		private RelayCommand _closeCommand;

		// Token: 0x040000B1 RID: 177
		private bool _isModified;

		// Token: 0x040000B2 RID: 178
		private FrameworkElement _ui;

		// Token: 0x040000B3 RID: 179
		private ImageSource _iconSource;
	}
}
