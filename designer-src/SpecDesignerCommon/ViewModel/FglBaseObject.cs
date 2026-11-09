using System;
using System.ComponentModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000046 RID: 70
	public class FglBaseObject : INotifyPropertyChanged, IEditableObject
	{
		// Token: 0x06000238 RID: 568 RVA: 0x00009D74 File Offset: 0x00007F74
		public void BeginEdit()
		{
		}

		// Token: 0x06000239 RID: 569 RVA: 0x00009D76 File Offset: 0x00007F76
		public void CancelEdit()
		{
			this.OnPropertyChanged("");
		}

		// Token: 0x0600023A RID: 570 RVA: 0x00009D83 File Offset: 0x00007F83
		public void EndEdit()
		{
			this.OnPropertyChanged("");
		}

		// Token: 0x14000007 RID: 7
		// (add) Token: 0x0600023B RID: 571 RVA: 0x00009D90 File Offset: 0x00007F90
		// (remove) Token: 0x0600023C RID: 572 RVA: 0x00009DC8 File Offset: 0x00007FC8
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600023D RID: 573 RVA: 0x00009DFD File Offset: 0x00007FFD
		protected void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}
	}
}
