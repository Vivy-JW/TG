using System;
using System.ComponentModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000F3 RID: 243
	public class ProgressBarViewModel : INotifyPropertyChanged
	{
		// Token: 0x06000821 RID: 2081 RVA: 0x00023FFA File Offset: 0x000221FA
		public ProgressBarViewModel()
		{
			this.Model = new ProgressBarModel();
		}

		// Token: 0x1700023F RID: 575
		// (get) Token: 0x06000822 RID: 2082 RVA: 0x0002400D File Offset: 0x0002220D
		// (set) Token: 0x06000823 RID: 2083 RVA: 0x00024015 File Offset: 0x00022215
		private ProgressBarModel Model { get; set; }

		// Token: 0x1400001E RID: 30
		// (add) Token: 0x06000824 RID: 2084 RVA: 0x00024020 File Offset: 0x00022220
		// (remove) Token: 0x06000825 RID: 2085 RVA: 0x00024058 File Offset: 0x00022258
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000826 RID: 2086 RVA: 0x0002408D File Offset: 0x0002228D
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x17000240 RID: 576
		// (get) Token: 0x06000827 RID: 2087 RVA: 0x000240A9 File Offset: 0x000222A9
		// (set) Token: 0x06000828 RID: 2088 RVA: 0x000240B6 File Offset: 0x000222B6
		public string Message
		{
			get
			{
				return this.Model.Message;
			}
			set
			{
				this.Model.Message = value;
				this.NotifyPropertyChanged("Message");
			}
		}

		// Token: 0x040002DC RID: 732
		public static ProgressBarViewModel Instance = new ProgressBarViewModel();
	}
}
