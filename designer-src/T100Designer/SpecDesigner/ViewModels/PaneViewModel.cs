using System;
using System.ComponentModel;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000008 RID: 8
	public class PaneViewModel : INotifyPropertyChanged
	{
		// Token: 0x17000014 RID: 20
		// (get) Token: 0x06000047 RID: 71 RVA: 0x00002DEF File Offset: 0x00000FEF
		// (set) Token: 0x06000048 RID: 72 RVA: 0x00002DF7 File Offset: 0x00000FF7
		public string Title
		{
			get
			{
				return this._title;
			}
			set
			{
				if (this._title != value)
				{
					this._title = value;
					this.RaisePropertyChanged("Title");
				}
			}
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x06000049 RID: 73 RVA: 0x00002E19 File Offset: 0x00001019
		// (set) Token: 0x0600004A RID: 74 RVA: 0x00002E21 File Offset: 0x00001021
		public string ContentId
		{
			get
			{
				return this._contentId;
			}
			set
			{
				if (this._contentId != value)
				{
					this._contentId = value;
					this.RaisePropertyChanged("ContentId");
				}
			}
		}

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x0600004B RID: 75 RVA: 0x00002E43 File Offset: 0x00001043
		// (set) Token: 0x0600004C RID: 76 RVA: 0x00002E4B File Offset: 0x0000104B
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				if (this._isSelected != value)
				{
					this._isSelected = value;
					this.RaisePropertyChanged("IsSelected");
				}
			}
		}

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x0600004D RID: 77 RVA: 0x00002E68 File Offset: 0x00001068
		// (set) Token: 0x0600004E RID: 78 RVA: 0x00002E70 File Offset: 0x00001070
		public bool IsActive
		{
			get
			{
				return this._isActive;
			}
			set
			{
				if (this._isActive != value)
				{
					this._isActive = value;
					this.RaisePropertyChanged("IsActive");
				}
			}
		}

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x0600004F RID: 79 RVA: 0x00002E8D File Offset: 0x0000108D
		// (set) Token: 0x06000050 RID: 80 RVA: 0x00002E95 File Offset: 0x00001095
		public bool IsVisible
		{
			get
			{
				return this._isVisible;
			}
			set
			{
				this._isVisible = value;
				this.RaisePropertyChanged("IsVisible");
			}
		}

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x06000051 RID: 81 RVA: 0x00002EAC File Offset: 0x000010AC
		// (remove) Token: 0x06000052 RID: 82 RVA: 0x00002EE4 File Offset: 0x000010E4
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000053 RID: 83 RVA: 0x00002F19 File Offset: 0x00001119
		public void RaisePropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x04000024 RID: 36
		private string _title;

		// Token: 0x04000025 RID: 37
		private string _contentId;

		// Token: 0x04000026 RID: 38
		private bool _isSelected;

		// Token: 0x04000027 RID: 39
		private bool _isActive;

		// Token: 0x04000028 RID: 40
		private bool _isVisible;
	}
}
