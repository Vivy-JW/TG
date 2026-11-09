using System;
using System.ComponentModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000095 RID: 149
	public class DSColumnViewModel : DSColumnModel, INotifyPropertyChanged
	{
		// Token: 0x14000017 RID: 23
		// (add) Token: 0x0600060D RID: 1549 RVA: 0x0001B904 File Offset: 0x00019B04
		// (remove) Token: 0x0600060E RID: 1550 RVA: 0x0001B93C File Offset: 0x00019B3C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600060F RID: 1551 RVA: 0x0001B971 File Offset: 0x00019B71
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x170001BC RID: 444
		// (get) Token: 0x06000610 RID: 1552 RVA: 0x0001B98D File Offset: 0x00019B8D
		// (set) Token: 0x06000611 RID: 1553 RVA: 0x0001B995 File Offset: 0x00019B95
		public new bool IsUsed
		{
			get
			{
				return base.IsUsed;
			}
			set
			{
				base.IsUsed = value;
				this.NotifyPropertyChanged("IsUsed");
			}
		}
	}
}
