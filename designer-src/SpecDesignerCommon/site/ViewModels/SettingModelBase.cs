using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices.ComTypes;
using System.Windows;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000032 RID: 50
	public abstract class SettingModelBase : INotifyPropertyChanged
	{
		// Token: 0x0600017A RID: 378 RVA: 0x00007B11 File Offset: 0x00005D11
		public SettingModelBase()
		{
		}

		// Token: 0x1700005A RID: 90
		// (get) Token: 0x0600017B RID: 379 RVA: 0x00007B20 File Offset: 0x00005D20
		// (set) Token: 0x0600017C RID: 380 RVA: 0x00007B28 File Offset: 0x00005D28
		public int Order
		{
			get
			{
				return this._order;
			}
			set
			{
				this._order = value;
				this.NotifyPropertyChanged("Order");
			}
		}

		// Token: 0x0600017D RID: 381 RVA: 0x00007B3C File Offset: 0x00005D3C
		internal static int ParseOrder(XElement element)
		{
			int num = -1;
			if (element.Attribute("order") != null)
			{
				int.TryParse(element.Attribute("order").Value, out num);
			}
			return num;
		}

		// Token: 0x1700005B RID: 91
		// (get) Token: 0x0600017E RID: 382 RVA: 0x00007B7B File Offset: 0x00005D7B
		// (set) Token: 0x0600017F RID: 383 RVA: 0x00007B83 File Offset: 0x00005D83
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				this._name = value;
				this.IsModifying = false;
				this.NotifyPropertyChanged("Name");
			}
		}

		// Token: 0x1700005C RID: 92
		// (get) Token: 0x06000180 RID: 384 RVA: 0x00007B9E File Offset: 0x00005D9E
		// (set) Token: 0x06000181 RID: 385 RVA: 0x00007BA6 File Offset: 0x00005DA6
		public SettingFolderModel Parent { get; set; }

		// Token: 0x1700005D RID: 93
		// (get) Token: 0x06000182 RID: 386 RVA: 0x00007BAF File Offset: 0x00005DAF
		public bool IsFolder
		{
			get
			{
				return this is SettingFolderModel;
			}
		}

		// Token: 0x1700005E RID: 94
		// (get) Token: 0x06000183 RID: 387 RVA: 0x00007BBA File Offset: 0x00005DBA
		// (set) Token: 0x06000184 RID: 388 RVA: 0x00007BC2 File Offset: 0x00005DC2
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				this._isSelected = value;
				this.NotifyPropertyChanged("IsSelected");
			}
		}

		// Token: 0x1700005F RID: 95
		// (get) Token: 0x06000185 RID: 389 RVA: 0x00007BD6 File Offset: 0x00005DD6
		// (set) Token: 0x06000186 RID: 390 RVA: 0x00007BDE File Offset: 0x00005DDE
		public bool IsModifying
		{
			get
			{
				return this._isModifying;
			}
			set
			{
				this._isModifying = value;
				this.NotifyPropertyChanged("IsModifying");
			}
		}

		// Token: 0x17000060 RID: 96
		// (get) Token: 0x06000187 RID: 391 RVA: 0x00007BF2 File Offset: 0x00005DF2
		public bool IsExpanded
		{
			get
			{
				return true;
			}
		}

		// Token: 0x06000188 RID: 392 RVA: 0x00007BF5 File Offset: 0x00005DF5
		public virtual void Remove()
		{
			if (this.Parent != null)
			{
				this.Parent.Remove(this);
			}
		}

		// Token: 0x17000061 RID: 97
		// (get) Token: 0x06000189 RID: 393 RVA: 0x00007C1C File Offset: 0x00005E1C
		public ICommand DeleteCommand
		{
			get
			{
				if (this._deleteCommand == null)
				{
					this._deleteCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteDelete();
					}, (object p) => this.CanDelete());
				}
				return this._deleteCommand;
			}
		}

		// Token: 0x0600018A RID: 394 RVA: 0x00007C68 File Offset: 0x00005E68
		public bool CanDelete()
		{
			return this.Parent != null;
		}

		// Token: 0x0600018B RID: 395 RVA: 0x00007C75 File Offset: 0x00005E75
		public void ExecuteDelete()
		{
			if (DesignerMessageBox.Show("Delete ?", "Delete", MessageBoxButton.YesNo) == MessageBoxResult.Yes)
			{
				this.Remove();
			}
		}

		// Token: 0x17000062 RID: 98
		// (get) Token: 0x0600018C RID: 396 RVA: 0x00007C98 File Offset: 0x00005E98
		public ICommand CancelNamingCommand
		{
			get
			{
				if (this._cancelNamingCommand == null)
				{
					this._cancelNamingCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteCancelNaming();
					});
				}
				return this._cancelNamingCommand;
			}
		}

		// Token: 0x0600018D RID: 397 RVA: 0x00007CD1 File Offset: 0x00005ED1
		public void ExecuteCancelNaming()
		{
			this.Name = this._name;
		}

		// Token: 0x17000063 RID: 99
		// (get) Token: 0x0600018E RID: 398 RVA: 0x00007CE8 File Offset: 0x00005EE8
		public ICommand CreateConnectionCommand
		{
			get
			{
				if (this._createConnectionCommand == null)
				{
					this._createConnectionCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteCreateConnection();
					});
				}
				return this._createConnectionCommand;
			}
		}

		// Token: 0x0600018F RID: 399 RVA: 0x00007D24 File Offset: 0x00005F24
		public void ExecuteCreateConnection()
		{
			this.IsSelected = false;
			SettingModel settingModel = SettingModel.Create();
			if (this is SettingFolderModel)
			{
				(this as SettingFolderModel).Insert(settingModel);
				settingModel.IsSelected = true;
				return;
			}
			if (this is SettingModel)
			{
				SettingFolderModel parent = this.Parent;
				if (this.Parent == null)
				{
					return;
				}
				parent.Insert(settingModel);
				settingModel.IsSelected = true;
			}
		}

		// Token: 0x17000064 RID: 100
		// (get) Token: 0x06000190 RID: 400 RVA: 0x00007D88 File Offset: 0x00005F88
		public ICommand CreateFolderCommand
		{
			get
			{
				if (this._createFolderCommand == null)
				{
					this._createFolderCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteCreateFolder();
					});
				}
				return this._createFolderCommand;
			}
		}

		// Token: 0x06000191 RID: 401 RVA: 0x00007DC4 File Offset: 0x00005FC4
		public void ExecuteCreateFolder()
		{
			this.IsSelected = false;
			SettingFolderModel settingFolderModel = SettingFolderModel.Create();
			if (this is SettingFolderModel)
			{
				(this as SettingFolderModel).Insert(settingFolderModel);
				settingFolderModel.IsSelected = true;
				return;
			}
			if (this is SettingModel)
			{
				SettingFolderModel parent = this.Parent;
				if (this.Parent == null)
				{
					return;
				}
				parent.Insert(settingFolderModel);
				settingFolderModel.IsSelected = true;
			}
		}

		// Token: 0x17000065 RID: 101
		// (get) Token: 0x06000192 RID: 402 RVA: 0x00007E30 File Offset: 0x00006030
		public ICommand ModifyCommand
		{
			get
			{
				if (this._modifyCommand == null)
				{
					this._modifyCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteModify();
					}, (object p) => this.CanModify());
				}
				return this._modifyCommand;
			}
		}

		// Token: 0x06000193 RID: 403 RVA: 0x00007E7C File Offset: 0x0000607C
		public bool CanModify()
		{
			return this.Parent != null;
		}

		// Token: 0x06000194 RID: 404 RVA: 0x00007E89 File Offset: 0x00006089
		public void ExecuteModify()
		{
			this.IsModifying = true;
		}

		// Token: 0x17000066 RID: 102
		// (get) Token: 0x06000195 RID: 405 RVA: 0x00007EA4 File Offset: 0x000060A4
		public ICommand CopyCommand
		{
			get
			{
				if (this._copyCommand == null)
				{
					this._copyCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteCopy();
					}, (object p) => this.CanCopy());
				}
				return this._copyCommand;
			}
		}

		// Token: 0x06000196 RID: 406 RVA: 0x00007EF0 File Offset: 0x000060F0
		public bool CanCopy()
		{
			return this is SettingModel;
		}

		// Token: 0x06000197 RID: 407 RVA: 0x00007F00 File Offset: 0x00006100
		public void ExecuteCopy()
		{
			if (!(this is SettingModel))
			{
				return;
			}
			SettingModel settingModel = (this as SettingModel).Clone() as SettingModel;
			settingModel.UID = Guid.NewGuid().ToString();
			settingModel.Parent = this.Parent;
			this.Parent.Insert(settingModel);
			settingModel.IsSelected = true;
		}

		// Token: 0x17000067 RID: 103
		// (get) Token: 0x06000198 RID: 408 RVA: 0x00007F6F File Offset: 0x0000616F
		public RelayCommand CeateDesktopLinkCommand
		{
			get
			{
				if (this._createDesktopLinkCommand == null)
				{
					this._createDesktopLinkCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteDesktopLink();
					}, (object p) => this.CanExecuteCreateDesktopLink());
				}
				return this._createDesktopLinkCommand;
			}
		}

		// Token: 0x06000199 RID: 409 RVA: 0x00007FA4 File Offset: 0x000061A4
		public bool CanExecuteCreateDesktopLink()
		{
			if (this is SettingModel)
			{
				try
				{
					RegistryReader.GetExeDir((this as SettingModel).Connection.Version);
				}
				catch
				{
					return false;
				}
				return true;
			}
			return false;
		}

		// Token: 0x0600019A RID: 410 RVA: 0x00007FEC File Offset: 0x000061EC
		private void ExecuteDesktopLink()
		{
			SettingModel settingModel = this as SettingModel;
			string text = null;
			try
			{
				text = RegistryReader.GetExeDir(settingModel.Connection.Version);
			}
			catch
			{
				return;
			}
			string folderPath = Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory);
			IShellLink shellLink = (IShellLink)new ShellLink();
			shellLink.SetDescription("Launch SpecDesigner");
			shellLink.SetPath(text);
			shellLink.SetArguments(settingModel.UID);
			IPersistFile persistFile = (IPersistFile)shellLink;
			persistFile.Save(folderPath + "\\" + this.Name + ".lnk", false);
			DesignerMessageBox.Show(Application.Current.FindResource("Message_CreateLinkFinish") as string);
		}

		// Token: 0x17000068 RID: 104
		// (get) Token: 0x0600019B RID: 411 RVA: 0x000080AC File Offset: 0x000062AC
		public ICommand MoveToPreviousCommand
		{
			get
			{
				if (this._moveToPreviousCommand == null)
				{
					this._moveToPreviousCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteMoveToPrevious();
					}, (object p) => this.CanMoveToPrevious());
				}
				return this._copyCommand;
			}
		}

		// Token: 0x0600019C RID: 412 RVA: 0x000080F8 File Offset: 0x000062F8
		public bool CanMoveToPrevious()
		{
			return this.Order > 0;
		}

		// Token: 0x0600019D RID: 413 RVA: 0x00008103 File Offset: 0x00006303
		public void ExecuteMoveToPrevious()
		{
			this.Parent.Insert(this, this.Order - 1);
		}

		// Token: 0x17000069 RID: 105
		// (get) Token: 0x0600019E RID: 414 RVA: 0x0000812C File Offset: 0x0000632C
		public ICommand MoveToNextCommand
		{
			get
			{
				if (this._moveToNextCommand == null)
				{
					this._moveToNextCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteMoveToNext();
					}, (object p) => this.CanMoveToNext());
				}
				return this._copyCommand;
			}
		}

		// Token: 0x0600019F RID: 415 RVA: 0x00008178 File Offset: 0x00006378
		public bool CanMoveToNext()
		{
			return this.Order > 0;
		}

		// Token: 0x060001A0 RID: 416 RVA: 0x00008183 File Offset: 0x00006383
		public void ExecuteMoveToNext()
		{
			this.Parent.Insert(this, this.Order + 1);
		}

		// Token: 0x060001A1 RID: 417
		public abstract SettingModelBase Clone();

		// Token: 0x060001A2 RID: 418
		public abstract XElement ToXML();

		// Token: 0x060001A3 RID: 419
		public abstract SettingModel GetSettingByUid(string uid);

		// Token: 0x14000006 RID: 6
		// (add) Token: 0x060001A4 RID: 420 RVA: 0x0000819C File Offset: 0x0000639C
		// (remove) Token: 0x060001A5 RID: 421 RVA: 0x000081D4 File Offset: 0x000063D4
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060001A6 RID: 422 RVA: 0x00008209 File Offset: 0x00006409
		internal void NotifyPropertyChanged(string info)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(info));
			}
		}

		// Token: 0x060001A7 RID: 423 RVA: 0x00008228 File Offset: 0x00006428
		internal bool CheckWorkspaceIsUsed(string dir, string uid)
		{
			if (!dir.EndsWith(Path.DirectorySeparatorChar.ToString()))
			{
				dir += Path.DirectorySeparatorChar;
			}
			if (!(this is SettingModel))
			{
				Collection<SettingModelBase> settings = (this as SettingFolderModel).Settings;
				foreach (SettingModelBase settingModelBase in settings)
				{
					bool flag = settingModelBase.CheckWorkspaceIsUsed(dir, uid);
					if (flag)
					{
						return true;
					}
				}
				return false;
			}
			SettingModel settingModel = this as SettingModel;
			if (settingModel.UID != uid)
			{
				string workspace = settingModel.Connection.Workspace;
				if (workspace != "" && settingModel.Connection.InWorkspace(dir))
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x04000098 RID: 152
		private string _name;

		// Token: 0x04000099 RID: 153
		private bool _isSelected;

		// Token: 0x0400009A RID: 154
		private bool _isModifying;

		// Token: 0x0400009B RID: 155
		private bool _isInstalled;

		// Token: 0x0400009C RID: 156
		private RelayCommand _deleteCommand;

		// Token: 0x0400009D RID: 157
		private RelayCommand _createConnectionCommand;

		// Token: 0x0400009E RID: 158
		private RelayCommand _createFolderCommand;

		// Token: 0x0400009F RID: 159
		private RelayCommand _modifyCommand;

		// Token: 0x040000A0 RID: 160
		private RelayCommand _cancelNamingCommand;

		// Token: 0x040000A1 RID: 161
		private RelayCommand _copyCommand;

		// Token: 0x040000A2 RID: 162
		private RelayCommand _createDesktopLinkCommand;

		// Token: 0x040000A3 RID: 163
		private RelayCommand _moveToPreviousCommand;

		// Token: 0x040000A4 RID: 164
		private RelayCommand _moveToNextCommand;

		// Token: 0x040000A5 RID: 165
		private int _order = -1;
	}
}
