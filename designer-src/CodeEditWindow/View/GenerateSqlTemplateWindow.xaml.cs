using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000034 RID: 52
	public partial class GenerateSqlTemplateWindow : Window
	{
		// Token: 0x1700004B RID: 75
		// (get) Token: 0x0600020A RID: 522 RVA: 0x000103D1 File Offset: 0x0000E5D1
		public static GenerateSqlTemplateWindow This
		{
			get
			{
				if (GenerateSqlTemplateWindow._this == null)
				{
					GenerateSqlTemplateWindow._this = new GenerateSqlTemplateWindow();
				}
				return GenerateSqlTemplateWindow._this;
			}
		}

		// Token: 0x1700004C RID: 76
		// (get) Token: 0x0600020B RID: 523 RVA: 0x000103E9 File Offset: 0x0000E5E9
		// (set) Token: 0x0600020C RID: 524 RVA: 0x000103F1 File Offset: 0x0000E5F1
		public string SqlType
		{
			get
			{
				return this.sqlType;
			}
			set
			{
				this.sqlType = value;
			}
		}

		// Token: 0x1700004D RID: 77
		// (get) Token: 0x0600020D RID: 525 RVA: 0x000103FA File Offset: 0x0000E5FA
		// (set) Token: 0x0600020E RID: 526 RVA: 0x00010402 File Offset: 0x0000E602
		public string MainTable
		{
			get
			{
				return this.mainTable;
			}
			set
			{
				this.mainTable = value;
			}
		}

		// Token: 0x1700004E RID: 78
		// (get) Token: 0x0600020F RID: 527 RVA: 0x0001040B File Offset: 0x0000E60B
		// (set) Token: 0x06000210 RID: 528 RVA: 0x00010413 File Offset: 0x0000E613
		public string JoinTable
		{
			get
			{
				return this.joinTable;
			}
			set
			{
				this.joinTable = value;
			}
		}

		// Token: 0x06000211 RID: 529 RVA: 0x0001041C File Offset: 0x0000E61C
		public GenerateSqlTemplateWindow()
		{
			this.InitializeComponent();
			this.Init();
			Application.Current.Exit += this.Current_Exit;
		}

		// Token: 0x06000212 RID: 530 RVA: 0x00010480 File Offset: 0x0000E680
		private void cbDisplay_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			ComboBox comboBox = sender as ComboBox;
			if (comboBox.SelectedIndex == 0)
			{
				this.txtVariableName.IsEnabled = false;
				return;
			}
			this.txtVariableName.IsEnabled = true;
		}

		// Token: 0x06000213 RID: 531 RVA: 0x000104CC File Offset: 0x0000E6CC
		private void Init()
		{
			List<string> list = (from t in TableColumnHelper.GetTables()
				select t.Attribute("name").Value).ToList<string>();
			this.cbMainTable.ItemsSource = list;
			this.cbJoinTable.ItemsSource = list;
		}

		// Token: 0x06000214 RID: 532 RVA: 0x00010520 File Offset: 0x0000E720
		private void cbSQLType_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			ComboBox comboBox = sender as ComboBox;
			if (comboBox.SelectedIndex == 0)
			{
				this.lblJoinTable.Visibility = Visibility.Visible;
				this.cbJoinTable.Visibility = Visibility.Visible;
				this.cbJoinType.Visibility = Visibility.Visible;
				this.btnAdd.Visibility = Visibility.Visible;
				this.btnDelete.Visibility = Visibility.Visible;
				this.dataGrid1.Visibility = Visibility.Visible;
			}
			else
			{
				this.lblJoinTable.Visibility = Visibility.Collapsed;
				this.cbJoinTable.Visibility = Visibility.Collapsed;
				this.cbJoinType.Visibility = Visibility.Collapsed;
				this.btnAdd.Visibility = Visibility.Collapsed;
				this.btnDelete.Visibility = Visibility.Collapsed;
				this.dataGrid1.Visibility = Visibility.Collapsed;
				this.joinTable = string.Empty;
				this.dataGrid1.ItemsSource = null;
				this.TableDetail = new ObservableCollection<JoinTableDetail>();
			}
			ComboBoxItem comboBoxItem = (ComboBoxItem)this.cbSQLType.SelectedItem;
			this.sqlType = comboBoxItem.Content.ToString();
			this.tbPreviewSQL.Text = this.setPreviewSQLContent(this.sqlType, this.MainTable, this.JoinTable);
		}

		// Token: 0x06000215 RID: 533 RVA: 0x00010638 File Offset: 0x0000E838
		private string setPreviewSQLContent(string sqlType, string mainTable, string joinTable)
		{
			string text = string.Empty;
			if (!string.IsNullOrEmpty(sqlType))
			{
				if (sqlType == "update")
				{
					text = sqlType.ToUpper() + " ";
					if (!string.IsNullOrEmpty(mainTable))
					{
						text = text + mainTable + " \r\nSET() \r\nVALUES() \r\n";
					}
				}
				else
				{
					text = sqlType.ToUpper() + " ";
					if (!string.IsNullOrEmpty(mainTable))
					{
						text = text + "\r\nFROM " + mainTable + "\r\n";
					}
					if (!string.IsNullOrEmpty(joinTable))
					{
						foreach (string text2 in joinTable.Split(new char[] { ';' }))
						{
							string[] array2 = text2.Split(new char[] { ':' });
							string text3 = text;
							text = string.Concat(new string[]
							{
								text3,
								array2[1],
								" ",
								array2[0],
								" ON "
							});
							XElement fk = TableColumnHelper.GetFK(mainTable, array2[0]);
							if (fk != null)
							{
								string[] array3 = fk.Attribute("fk_detail").Value.Split(new char[] { ',' });
								string[] array4 = fk.Attribute("fk_master").Value.Split(new char[] { ',' });
								int num = -1;
								for (int j = 0; j < array3.Length; j++)
								{
									if (array3[j].Contains("ent"))
									{
										num = j;
									}
								}
								if (num != -1)
								{
									text = text + array3[num] + " = g_enterprise \r\n";
								}
								for (int k = 0; k < array3.Length; k++)
								{
									if (k != num)
									{
										string text4 = text;
										text = string.Concat(new string[]
										{
											text4,
											"AND ",
											array3[k],
											" = ",
											array4[k],
											"\r\n"
										});
									}
								}
							}
							else
							{
								text += " \r\n";
							}
						}
					}
				}
			}
			if (!string.IsNullOrEmpty(mainTable))
			{
				string[] array5 = TableColumnHelper.GetPK(mainTable).Split(new char[] { ',' });
				int num2 = -1;
				bool flag = false;
				for (int l = 0; l < array5.Length; l++)
				{
					if (array5[l].Contains("ent"))
					{
						num2 = l;
					}
				}
				text += "WHERE ";
				if (num2 != -1)
				{
					text = text + array5[num2] + " = g_enterprise ";
					flag = true;
				}
				for (int m = 0; m < array5.Length; m++)
				{
					if (m != num2)
					{
						if (flag)
						{
							text += "\r\n";
						}
						text = text + "AND " + array5[m] + " = ";
						flag = true;
					}
				}
			}
			return text;
		}

		// Token: 0x06000216 RID: 534 RVA: 0x00010922 File Offset: 0x0000EB22
		private void Current_Exit(object sender, ExitEventArgs e)
		{
			Application.Current.Exit -= this.Current_Exit;
			base.Close();
		}

		// Token: 0x06000217 RID: 535 RVA: 0x00010940 File Offset: 0x0000EB40
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x06000218 RID: 536 RVA: 0x00010970 File Offset: 0x0000EB70
		private void btnAdd_Click(object sender, RoutedEventArgs e)
		{
			ComboBoxItem comboBoxItem = (ComboBoxItem)this.cbJoinType.SelectedItem;
			List<JoinTableDetail> list = this.TableDetail.Where<JoinTableDetail>((JoinTableDetail i) => i.TableName == this.cbJoinTable.SelectedItem.ToString()).ToList<JoinTableDetail>();
			if (list.Count > 0)
			{
				MessageBox.Show(Application.Current.FindResource("Message_TableIsExist") as string);
				return;
			}
			this.TableDetail.Add(new JoinTableDetail
			{
				TableName = this.cbJoinTable.SelectedItem.ToString(),
				JoinType = comboBoxItem.Content.ToString()
			});
			this.dataGrid1.ItemsSource = this.TableDetail;
			this.JoinTable = "";
			foreach (JoinTableDetail joinTableDetail in this.TableDetail.ToArray<JoinTableDetail>())
			{
				string text = this.JoinTable;
				this.JoinTable = string.Concat(new string[] { text, joinTableDetail.TableName, ":", joinTableDetail.JoinType, ";" });
			}
			this.JoinTable = this.JoinTable.Substring(0, this.JoinTable.Length - 1);
			this.tbPreviewSQL.Text = this.setPreviewSQLContent(this.SqlType, this.MainTable, this.JoinTable);
		}

		// Token: 0x06000219 RID: 537 RVA: 0x00010AD8 File Offset: 0x0000ECD8
		private void btnDelete_Click(object sender, RoutedEventArgs e)
		{
			foreach (JoinTableDetail joinTableDetail in this.TableDetail.ToArray<JoinTableDetail>())
			{
				if (joinTableDetail.IsSelected)
				{
					this.RemoveItem(this.TableDetail, joinTableDetail);
				}
			}
			this.JoinTable = "";
			foreach (JoinTableDetail joinTableDetail2 in this.TableDetail.ToArray<JoinTableDetail>())
			{
				string text = this.JoinTable;
				this.JoinTable = string.Concat(new string[] { text, joinTableDetail2.TableName, ":", joinTableDetail2.JoinType, ";" });
			}
			this.JoinTable = this.JoinTable.Substring(0, this.JoinTable.Length - 1);
			this.tbPreviewSQL.Text = this.setPreviewSQLContent(this.SqlType, this.MainTable, this.JoinTable);
		}

		// Token: 0x0600021A RID: 538 RVA: 0x00010BD4 File Offset: 0x0000EDD4
		private void cbMainTable_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.mainTable = this.cbMainTable.SelectedItem.ToString();
			this.tbPreviewSQL.Text = this.setPreviewSQLContent(this.SqlType, this.MainTable, this.JoinTable);
		}

		// Token: 0x0600021B RID: 539 RVA: 0x00010C0F File Offset: 0x0000EE0F
		private void copySQL_Click(object sender, RoutedEventArgs e)
		{
			Clipboard.SetText(this.tbPreviewSQL.Text);
		}

		// Token: 0x0600021C RID: 540 RVA: 0x00010C24 File Offset: 0x0000EE24
		private void testSQL_Click(object sender, RoutedEventArgs e)
		{
			if (!Directory.Exists("C:\\t100temp"))
			{
				Directory.CreateDirectory("C:\\t100temp");
			}
			string text = Guid.NewGuid().ToString() + ".sql";
			File.WriteAllText("C:\\t100temp\\" + text, this.tbPreviewSQL.Text);
			ConnectionManager.RunProgram(string.Format("adzi170 -SQL '{0}'", text));
		}

		// Token: 0x0600021D RID: 541 RVA: 0x00010C94 File Offset: 0x0000EE94
		private void Save_Click(object sender, RoutedEventArgs e)
		{
			string text = string.Empty;
			if (this.cbDisplay.Text == "SQL")
			{
				text = this.tbPreviewSQL.Text;
			}
			else
			{
				text = "LET " + this.txtVariableName.Text + " = ";
				string text2 = "".PadLeft(text.Length, ' ');
				string[] array = this.tbPreviewSQL.Text.Split(new string[] { "\r\n" }, StringSplitOptions.RemoveEmptyEntries);
				for (int i = 0; i < array.Length; i++)
				{
					if (i == 0)
					{
						text = text + "\"" + array[i] + "\",\r\n";
					}
					else if (i == array.Length - 1)
					{
						string text3 = text;
						text = string.Concat(new string[]
						{
							text3,
							text2,
							"\"",
							array[i],
							"\""
						});
					}
					else
					{
						string text4 = text;
						text = string.Concat(new string[]
						{
							text4,
							text2,
							"\"",
							array[i],
							"\",\r\n"
						});
					}
				}
			}
			EventAggregatorManager.Global.GetEvent<SaveSqlResultEvent>().Publish(text);
			Application.Current.Exit -= this.Current_Exit;
			base.Close();
		}

		// Token: 0x0600021E RID: 542 RVA: 0x00010E14 File Offset: 0x0000F014
		public void RemoveItem(ObservableCollection<JoinTableDetail> collection, JoinTableDetail instance)
		{
			collection.Remove(collection.Where<JoinTableDetail>((JoinTableDetail i) => i.TableName == instance.TableName).Single<JoinTableDetail>());
		}

		// Token: 0x040000DA RID: 218
		private static GenerateSqlTemplateWindow _this;

		// Token: 0x040000DB RID: 219
		private string sqlType = string.Empty;

		// Token: 0x040000DC RID: 220
		private string mainTable = string.Empty;

		// Token: 0x040000DD RID: 221
		private string joinTable = string.Empty;

		// Token: 0x040000DE RID: 222
		private ObservableCollection<JoinTableDetail> TableDetail = new ObservableCollection<JoinTableDetail>();
	}
}
