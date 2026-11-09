using System;
using System.ComponentModel;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000035 RID: 53
	public class JoinTableDetail : INotifyPropertyChanged
	{
		// Token: 0x1700004F RID: 79
		// (get) Token: 0x06000224 RID: 548 RVA: 0x000110AD File Offset: 0x0000F2AD
		// (set) Token: 0x06000225 RID: 549 RVA: 0x000110B5 File Offset: 0x0000F2B5
		public bool IsSelected
		{
			get
			{
				return this.isSelected;
			}
			set
			{
				this.isSelected = value;
				this.OnPropertyChanged("IsSelected");
			}
		}

		// Token: 0x17000050 RID: 80
		// (get) Token: 0x06000226 RID: 550 RVA: 0x000110C9 File Offset: 0x0000F2C9
		// (set) Token: 0x06000227 RID: 551 RVA: 0x000110D1 File Offset: 0x0000F2D1
		public string TableName
		{
			get
			{
				return this.tableName;
			}
			set
			{
				this.tableName = value;
				this.OnPropertyChanged("TableName");
			}
		}

		// Token: 0x17000051 RID: 81
		// (get) Token: 0x06000228 RID: 552 RVA: 0x000110E5 File Offset: 0x0000F2E5
		// (set) Token: 0x06000229 RID: 553 RVA: 0x000110ED File Offset: 0x0000F2ED
		public string JoinType
		{
			get
			{
				return this.joinType;
			}
			set
			{
				this.joinType = value;
				this.OnPropertyChanged("JoinType");
			}
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x0600022A RID: 554 RVA: 0x00011104 File Offset: 0x0000F304
		// (remove) Token: 0x0600022B RID: 555 RVA: 0x0001113C File Offset: 0x0000F33C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600022C RID: 556 RVA: 0x00011171 File Offset: 0x0000F371
		private void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x040000F5 RID: 245
		private bool isSelected;

		// Token: 0x040000F6 RID: 246
		private string tableName;

		// Token: 0x040000F7 RID: 247
		private string joinType;
	}
}
