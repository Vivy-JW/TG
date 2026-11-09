using System;
using System.ComponentModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000FB RID: 251
	public class AvalonOptions : INotifyPropertyChanged
	{
		// Token: 0x17000248 RID: 584
		// (get) Token: 0x06000833 RID: 2099 RVA: 0x000240F4 File Offset: 0x000222F4
		// (set) Token: 0x06000834 RID: 2100 RVA: 0x000240FC File Offset: 0x000222FC
		public bool IsActive
		{
			get
			{
				return this._IsActive;
			}
			set
			{
				this._IsActive = value;
				this.NotifyPropertyChanged("IsActive");
			}
		}

		// Token: 0x17000249 RID: 585
		// (get) Token: 0x06000835 RID: 2101 RVA: 0x00024110 File Offset: 0x00022310
		// (set) Token: 0x06000836 RID: 2102 RVA: 0x00024118 File Offset: 0x00022318
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

		// Token: 0x1700024A RID: 586
		// (get) Token: 0x06000837 RID: 2103 RVA: 0x0002412C File Offset: 0x0002232C
		// (set) Token: 0x06000838 RID: 2104 RVA: 0x00024134 File Offset: 0x00022334
		public bool IsVisible
		{
			get
			{
				return this._isVisible;
			}
			set
			{
				this._isVisible = value;
				this.NotifyPropertyChanged("IsVisible");
			}
		}

		// Token: 0x1400001F RID: 31
		// (add) Token: 0x06000839 RID: 2105 RVA: 0x00024148 File Offset: 0x00022348
		// (remove) Token: 0x0600083A RID: 2106 RVA: 0x00024180 File Offset: 0x00022380
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600083B RID: 2107 RVA: 0x000241B5 File Offset: 0x000223B5
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x040002E0 RID: 736
		private bool _IsActive;

		// Token: 0x040002E1 RID: 737
		private bool _isSelected;

		// Token: 0x040002E2 RID: 738
		private bool _isVisible;
	}
}
